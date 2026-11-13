-- Stockbook Desktop — Supabase backend
-- Replaces the local SQLite database. Identity comes from Supabase Auth
-- (auth.users); every business row is scoped to one stockbook_companies row.
-- All money is stored as integer paise (cents), matching the previous schema.

-- ------------------------------------------------------------------ tables

create table if not exists public.stockbook_companies (
  id bigint generated always as identity primary key,
  user_id uuid not null unique references auth.users (id) on delete cascade,
  name text not null,
  email text not null,
  created_at timestamptz not null default now()
);

create unique index if not exists stockbook_companies_email_idx
  on public.stockbook_companies (lower(email));

create table if not exists public.stockbook_products (
  id bigint generated always as identity primary key,
  company_id bigint not null references public.stockbook_companies (id) on delete cascade,
  name text not null,
  sku text,
  category text not null,
  purchase_price bigint not null default 0 check (purchase_price >= 0),
  sell_price bigint not null default 0 check (sell_price >= 0),
  quantity bigint not null default 0 check (quantity >= 0),
  low_stock_threshold bigint not null default 5 check (low_stock_threshold >= 0),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create unique index if not exists stockbook_products_company_sku_idx
  on public.stockbook_products (company_id, lower(sku))
  where sku is not null and sku <> '';

create index if not exists stockbook_products_company_name_idx
  on public.stockbook_products (company_id, lower(name));
create index if not exists stockbook_products_company_category_idx
  on public.stockbook_products (company_id, lower(category));

create table if not exists public.stockbook_purchases (
  id bigint generated always as identity primary key,
  company_id bigint not null references public.stockbook_companies (id) on delete cascade,
  product_id bigint not null references public.stockbook_products (id) on delete restrict,
  quantity bigint not null check (quantity > 0),
  unit_cost bigint not null check (unit_cost >= 0),
  total_cost bigint not null check (total_cost >= 0),
  purchased_at timestamptz not null,
  note text,
  created_at timestamptz not null default now()
);

create index if not exists stockbook_purchases_company_date_idx
  on public.stockbook_purchases (company_id, purchased_at);
create index if not exists stockbook_purchases_company_product_idx
  on public.stockbook_purchases (company_id, product_id);

create table if not exists public.stockbook_sales (
  id bigint generated always as identity primary key,
  company_id bigint not null references public.stockbook_companies (id) on delete cascade,
  product_id bigint not null references public.stockbook_products (id) on delete restrict,
  quantity bigint not null check (quantity > 0),
  unit_price bigint not null check (unit_price >= 0),
  total_price bigint not null check (total_price >= 0),
  unit_cost bigint not null check (unit_cost >= 0),
  total_cost bigint not null check (total_cost >= 0),
  profit bigint not null,
  payment_mode text not null default 'cash' check (payment_mode in ('cash', 'upi')),
  sold_at timestamptz not null,
  note text,
  created_at timestamptz not null default now()
);

create index if not exists stockbook_sales_company_date_idx
  on public.stockbook_sales (company_id, sold_at);
create index if not exists stockbook_sales_company_product_idx
  on public.stockbook_sales (company_id, product_id);

-- ------------------------------------------------------------- identity helpers

create or replace function public.stockbook_company_id()
returns bigint
language sql
stable
security definer
set search_path = public
as $$
  select id from public.stockbook_companies where user_id = auth.uid();
$$;

create or replace function public.stockbook_escape_like(p_value text)
returns text
language sql
immutable
as $$
  select replace(replace(replace(coalesce(p_value, ''), '\', '\\'), '%', '\%'), '_', '\_');
$$;

-- Creates the workspace row for the signed-in auth user, or returns the
-- existing one. Sign-up passes the company name from the auth metadata.
create or replace function public.stockbook_bootstrap_company(p_name text default null)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_user uuid := auth.uid();
  v_email text;
  v_meta_name text;
  v_name text;
  v_row public.stockbook_companies%rowtype;
begin
  if v_user is null then
    raise exception 'unauthorized';
  end if;

  select email,
         nullif(btrim(coalesce(raw_user_meta_data->>'company_name', '')), '')
  into v_email, v_meta_name
  from auth.users where id = v_user;
  if v_email is null then
    raise exception 'unauthorized';
  end if;

  select * into v_row from public.stockbook_companies where user_id = v_user;
  if found then
    return jsonb_build_object('id', v_row.id, 'name', v_row.name, 'email', v_row.email);
  end if;

  v_name := nullif(btrim(coalesce(p_name, '')), '');
  if v_name is null then
    v_name := v_meta_name;
  end if;
  if v_name is null then
    v_name := split_part(v_email, '@', 1);
  end if;

  insert into public.stockbook_companies (user_id, name, email)
  values (v_user, v_name, v_email)
  on conflict (user_id) do nothing
  returning * into v_row;

  if v_row.id is null then
    select * into v_row from public.stockbook_companies where user_id = v_user;
  end if;

  return jsonb_build_object('id', v_row.id, 'name', v_row.name, 'email', v_row.email);
end;
$$;

-- ---------------------------------------------------------------- categories

create or replace function public.stockbook_categories()
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  return coalesce((
    select jsonb_agg(category order by lower(category))
    from (
      select min(category) as category
      from public.stockbook_products
      where company_id = v_company
      group by lower(category)
    ) grouped
  ), '[]'::jsonb);
end;
$$;

-- ----------------------------------------------------------------- overview

create or replace function public.stockbook_overview(
  p_from timestamptz,
  p_to timestamptz,
  p_all_time boolean,
  p_category text,
  p_tz_offset integer
)
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_offset integer := coalesce(p_tz_offset, 0);
  v_sales jsonb;
  v_purchases jsonb;
  v_inventory jsonb;
  v_trend jsonb;
  v_top jsonb;
  v_low jsonb;
  v_recent jsonb;
  v_categories jsonb;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  v_categories := public.stockbook_categories();

  select jsonb_build_object(
    'revenue', coalesce(sum(s.total_price), 0)::bigint,
    'cost', coalesce(sum(s.total_cost), 0)::bigint,
    'profit', coalesce(sum(s.profit), 0)::bigint,
    'units', coalesce(sum(s.quantity), 0)::bigint,
    'cashRevenue', coalesce(sum(case when s.payment_mode = 'cash' then s.total_price else 0 end), 0)::bigint,
    'upiRevenue', coalesce(sum(case when s.payment_mode = 'upi' then s.total_price else 0 end), 0)::bigint,
    'cashCount', coalesce(sum(case when s.payment_mode = 'cash' then 1 else 0 end), 0)::bigint,
    'upiCount', coalesce(sum(case when s.payment_mode = 'upi' then 1 else 0 end), 0)::bigint,
    'count', count(*)
  )
  into v_sales
  from public.stockbook_sales s
  join public.stockbook_products p on p.id = s.product_id
  where s.company_id = v_company
    and (p_all_time or (
      s.sold_at >= coalesce(p_from, '-infinity'::timestamptz)
      and s.sold_at <= coalesce(p_to, 'infinity'::timestamptz)
    ))
    and (p_category is null or lower(p.category) = lower(p_category));

  select jsonb_build_object(
    'spend', coalesce(sum(pu.total_cost), 0)::bigint,
    'units', coalesce(sum(pu.quantity), 0)::bigint,
    'count', count(*)
  )
  into v_purchases
  from public.stockbook_purchases pu
  join public.stockbook_products p on p.id = pu.product_id
  where pu.company_id = v_company
    and (p_all_time or (
      pu.purchased_at >= coalesce(p_from, '-infinity'::timestamptz)
      and pu.purchased_at <= coalesce(p_to, 'infinity'::timestamptz)
    ))
    and (p_category is null or lower(p.category) = lower(p_category));

  select jsonb_build_object(
    'units', coalesce(sum(quantity), 0)::bigint,
    'costValue', coalesce(sum(quantity * purchase_price), 0)::bigint,
    'products', count(*)
  )
  into v_inventory
  from public.stockbook_products
  where company_id = v_company;

  select coalesce(jsonb_agg(jsonb_build_object(
      'date', day,
      'revenue', revenue,
      'units', units
    ) order by day), '[]'::jsonb)
  into v_trend
  from (
    select ((s.sold_at at time zone 'UTC') + make_interval(mins => v_offset))::date::text as day,
           coalesce(sum(s.total_price), 0)::bigint as revenue,
           coalesce(sum(s.quantity), 0)::bigint as units
    from public.stockbook_sales s
    join public.stockbook_products p on p.id = s.product_id
    where s.company_id = v_company
      and (p_all_time or (
        s.sold_at >= coalesce(p_from, '-infinity'::timestamptz)
        and s.sold_at <= coalesce(p_to, 'infinity'::timestamptz)
      ))
      and (p_category is null or lower(p.category) = lower(p_category))
    group by 1
    order by 1
  ) days;

  select coalesce(jsonb_agg(jsonb_build_object(
      'id', t.product_id,
      'name', p.name,
      'category', p.category,
      'units', t.units,
      'revenue', t.revenue,
      'profit', t.profit
    ) order by t.revenue desc, t.units desc), '[]'::jsonb)
  into v_top
  from (
    select s.product_id,
           coalesce(sum(s.quantity), 0)::bigint as units,
           coalesce(sum(s.total_price), 0)::bigint as revenue,
           coalesce(sum(s.profit), 0)::bigint as profit
    from public.stockbook_sales s
    join public.stockbook_products p on p.id = s.product_id
    where s.company_id = v_company
      and (p_all_time or (
        s.sold_at >= coalesce(p_from, '-infinity'::timestamptz)
        and s.sold_at <= coalesce(p_to, 'infinity'::timestamptz)
      ))
      and (p_category is null or lower(p.category) = lower(p_category))
    group by s.product_id
    order by revenue desc, units desc
    limit 5
  ) t
  join public.stockbook_products p on p.id = t.product_id;

  select coalesce(jsonb_agg(jsonb_build_object(
      'id', x.id,
      'name', x.name,
      'quantity', x.quantity,
      'threshold', x.low_stock_threshold
    ) order by x.quantity asc, lower(x.name) asc), '[]'::jsonb)
  into v_low
  from (
    select id, name, quantity, low_stock_threshold
    from public.stockbook_products
    where company_id = v_company and quantity <= low_stock_threshold
    order by quantity asc, lower(name) asc
    limit 6
  ) x;

  select coalesce(jsonb_agg(jsonb_build_object(
      'id', x.id,
      'productName', x.product_name,
      'quantity', x.quantity,
      'totalPrice', x.total_price,
      'profit', x.profit,
      'paymentMode', x.payment_mode,
      'soldAt', x.sold_at
    ) order by x.sold_at desc, x.id desc), '[]'::jsonb)
  into v_recent
  from (
    select s.id, p.name as product_name, s.quantity, s.total_price,
           s.profit, s.payment_mode, s.sold_at
    from public.stockbook_sales s
    join public.stockbook_products p on p.id = s.product_id
    where s.company_id = v_company
      and (p_all_time or (
        s.sold_at >= coalesce(p_from, '-infinity'::timestamptz)
        and s.sold_at <= coalesce(p_to, 'infinity'::timestamptz)
      ))
      and (p_category is null or lower(p.category) = lower(p_category))
    order by s.sold_at desc, s.id desc
    limit 6
  ) x;

  return jsonb_build_object(
    'categories', v_categories,
    'sales', v_sales,
    'purchases', v_purchases,
    'inventory', v_inventory,
    'trend', v_trend,
    'topProducts', v_top,
    'lowStock', v_low,
    'recentSales', v_recent
  );
end;
$$;

-- -------------------------------------------------------------------- items

create or replace function public.stockbook_items(
  p_search text,
  p_category text,
  p_page integer,
  p_page_size integer
)
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_page_size integer := greatest(coalesce(p_page_size, 50), 1);
  v_count bigint;
  v_category_count bigint;
  v_avg_cost bigint;
  v_avg_sell bigint;
  v_pages integer;
  v_page integer;
  v_items jsonb;
  v_categories jsonb;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  select count(*),
         count(distinct lower(category)),
         coalesce(round(avg(purchase_price)), 0)::bigint,
         coalesce(round(avg(sell_price)), 0)::bigint
  into v_count, v_category_count, v_avg_cost, v_avg_sell
  from public.stockbook_products p
  where p.company_id = v_company
    and (p_search is null or p_search = '' or (
      p.name ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
      or coalesce(p.sku, '') ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
      or p.category ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
    ))
    and (p_category is null or p_category = '' or lower(p.category) = lower(p_category));

  v_pages := greatest(1, ceil(v_count::numeric / v_page_size)::integer);
  v_page := least(greatest(coalesce(p_page, 1), 1), v_pages);

  select coalesce(jsonb_agg(jsonb_build_object(
      'id', x.id,
      'name', x.name,
      'sku', x.sku,
      'category', x.category,
      'quantity', x.quantity,
      'purchasePrice', x.purchase_price,
      'sellPrice', x.sell_price,
      'lowStockThreshold', x.low_stock_threshold
    ) order by x.created_at desc, x.id desc), '[]'::jsonb)
  into v_items
  from (
    select p.*
    from public.stockbook_products p
    where p.company_id = v_company
      and (p_search is null or p_search = '' or (
        p.name ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
        or coalesce(p.sku, '') ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
        or p.category ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
      ))
      and (p_category is null or p_category = '' or lower(p.category) = lower(p_category))
    order by p.created_at desc, p.id desc
    limit v_page_size offset (v_page - 1) * v_page_size
  ) x;

  v_categories := public.stockbook_categories();

  return jsonb_build_object(
    'items', v_items,
    'summary', jsonb_build_object(
      'count', v_count,
      'categories', v_category_count,
      'avgCost', v_avg_cost,
      'avgSell', v_avg_sell
    ),
    'page', v_page,
    'totalPages', v_pages,
    'pageSize', v_page_size,
    'categories', v_categories
  );
end;
$$;

-- ---------------------------------------------------------------- inventory

create or replace function public.stockbook_inventory(
  p_search text,
  p_category text,
  p_low_stock boolean,
  p_page integer,
  p_page_size integer
)
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_page_size integer := greatest(coalesce(p_page_size, 50), 1);
  v_count bigint;
  v_units bigint;
  v_cost numeric;
  v_retail numeric;
  v_pages integer;
  v_page integer;
  v_products jsonb;
  v_categories jsonb;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  select count(*),
         coalesce(sum(quantity), 0),
         coalesce(sum(quantity * purchase_price), 0),
         coalesce(sum(quantity * sell_price), 0)
  into v_count, v_units, v_cost, v_retail
  from public.stockbook_products p
  where p.company_id = v_company
    and (p_search is null or p_search = '' or (
      p.name ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
      or coalesce(p.sku, '') ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
      or p.category ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
    ))
    and (p_category is null or p_category = '' or lower(p.category) = lower(p_category))
    and (not coalesce(p_low_stock, false) or quantity <= low_stock_threshold);

  v_pages := greatest(1, ceil(v_count::numeric / v_page_size)::integer);
  v_page := least(greatest(coalesce(p_page, 1), 1), v_pages);

  select coalesce(jsonb_agg(jsonb_build_object(
      'id', x.id,
      'name', x.name,
      'sku', x.sku,
      'category', x.category,
      'quantity', x.quantity,
      'purchasePrice', x.purchase_price,
      'sellPrice', x.sell_price,
      'lowStockThreshold', x.low_stock_threshold
    ) order by x.stock_rank asc, lower(x.name) asc), '[]'::jsonb)
  into v_products
  from (
    select p.*,
           case
             when p.quantity = 0 then 0
             when p.quantity <= p.low_stock_threshold then 1
             else 2
           end as stock_rank
    from public.stockbook_products p
    where p.company_id = v_company
      and (p_search is null or p_search = '' or (
        p.name ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
        or coalesce(p.sku, '') ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
        or p.category ilike '%' || public.stockbook_escape_like(p_search) || '%' escape '\'
      ))
      and (p_category is null or p_category = '' or lower(p.category) = lower(p_category))
      and (not coalesce(p_low_stock, false) or p.quantity <= p.low_stock_threshold)
    order by stock_rank asc, lower(p.name) asc
    limit v_page_size offset (v_page - 1) * v_page_size
  ) x;

  v_categories := public.stockbook_categories();

  return jsonb_build_object(
    'products', v_products,
    'summary', jsonb_build_object(
      'count', v_count,
      'units', v_units,
      'costValue', v_cost::bigint,
      'retailValue', v_retail::bigint
    ),
    'page', v_page,
    'totalPages', v_pages,
    'pageSize', v_page_size,
    'categories', v_categories
  );
end;
$$;

-- ----------------------------------------------------------------- purchases

create or replace function public.stockbook_purchases(
  p_from timestamptz,
  p_to timestamptz,
  p_all_time boolean,
  p_category text,
  p_page integer,
  p_page_size integer
)
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_page_size integer := greatest(coalesce(p_page_size, 25), 1);
  v_count bigint;
  v_spend bigint;
  v_units bigint;
  v_pages integer;
  v_page integer;
  v_rows jsonb;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  select count(*),
         coalesce(sum(pu.total_cost), 0)::bigint,
         coalesce(sum(pu.quantity), 0)::bigint
  into v_count, v_spend, v_units
  from public.stockbook_purchases pu
  join public.stockbook_products p on p.id = pu.product_id
  where pu.company_id = v_company
    and (p_all_time or (
      pu.purchased_at >= coalesce(p_from, '-infinity'::timestamptz)
      and pu.purchased_at <= coalesce(p_to, 'infinity'::timestamptz)
    ))
    and (p_category is null or p_category = '' or lower(p.category) = lower(p_category));

  v_pages := greatest(1, ceil(v_count::numeric / v_page_size)::integer);
  v_page := least(greatest(coalesce(p_page, 1), 1), v_pages);

  select coalesce(jsonb_agg(jsonb_build_object(
      'id', x.id,
      'productName', x.product_name,
      'sku', x.sku,
      'category', x.category,
      'quantity', x.quantity,
      'unitCost', x.unit_cost,
      'totalCost', x.total_cost,
      'purchasedAt', x.purchased_at,
      'note', x.note
    ) order by x.purchased_at desc, x.id desc), '[]'::jsonb)
  into v_rows
  from (
    select pu.id, p.name as product_name, p.sku, p.category,
           pu.quantity, pu.unit_cost, pu.total_cost, pu.purchased_at, pu.note
    from public.stockbook_purchases pu
    join public.stockbook_products p on p.id = pu.product_id
    where pu.company_id = v_company
      and (p_all_time or (
        pu.purchased_at >= coalesce(p_from, '-infinity'::timestamptz)
        and pu.purchased_at <= coalesce(p_to, 'infinity'::timestamptz)
      ))
      and (p_category is null or p_category = '' or lower(p.category) = lower(p_category))
    order by pu.purchased_at desc, pu.id desc
    limit v_page_size offset (v_page - 1) * v_page_size
  ) x;

  return jsonb_build_object(
    'rows', v_rows,
    'summary', jsonb_build_object('spend', v_spend, 'units', v_units, 'count', v_count),
    'page', v_page,
    'totalPages', v_pages,
    'pageSize', v_page_size,
    'categories', public.stockbook_categories()
  );
end;
$$;

-- --------------------------------------------------------------------- sales

create or replace function public.stockbook_sales(
  p_from timestamptz,
  p_to timestamptz,
  p_all_time boolean,
  p_category text,
  p_page integer,
  p_page_size integer
)
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_page_size integer := greatest(coalesce(p_page_size, 25), 1);
  v_count bigint;
  v_summary jsonb;
  v_pages integer;
  v_page integer;
  v_rows jsonb;
  v_breakdown jsonb;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  select jsonb_build_object(
      'revenue', coalesce(sum(s.total_price), 0)::bigint,
      'cost', coalesce(sum(s.total_cost), 0)::bigint,
      'profit', coalesce(sum(s.profit), 0)::bigint,
      'units', coalesce(sum(s.quantity), 0)::bigint,
      'cashRevenue', coalesce(sum(case when s.payment_mode = 'cash' then s.total_price else 0 end), 0)::bigint,
      'upiRevenue', coalesce(sum(case when s.payment_mode = 'upi' then s.total_price else 0 end), 0)::bigint,
      'cashCount', coalesce(sum(case when s.payment_mode = 'cash' then 1 else 0 end), 0)::bigint,
      'upiCount', coalesce(sum(case when s.payment_mode = 'upi' then 1 else 0 end), 0)::bigint,
      'count', count(*)
    ),
    count(*)
  into v_summary, v_count
  from public.stockbook_sales s
  join public.stockbook_products p on p.id = s.product_id
  where s.company_id = v_company
    and (p_all_time or (
      s.sold_at >= coalesce(p_from, '-infinity'::timestamptz)
      and s.sold_at <= coalesce(p_to, 'infinity'::timestamptz)
    ))
    and (p_category is null or p_category = '' or lower(p.category) = lower(p_category));

  v_pages := greatest(1, ceil(v_count::numeric / v_page_size)::integer);
  v_page := least(greatest(coalesce(p_page, 1), 1), v_pages);

  select coalesce(jsonb_agg(jsonb_build_object(
      'id', x.id,
      'productName', x.product_name,
      'category', x.category,
      'quantity', x.quantity,
      'unitPrice', x.unit_price,
      'totalPrice', x.total_price,
      'unitCost', x.unit_cost,
      'profit', x.profit,
      'paymentMode', x.payment_mode,
      'soldAt', x.sold_at,
      'note', x.note
    ) order by x.sold_at desc, x.id desc), '[]'::jsonb)
  into v_rows
  from (
    select s.id, p.name as product_name, p.category,
           s.quantity, s.unit_price, s.total_price,
           s.unit_cost, s.profit, s.payment_mode, s.sold_at, s.note
    from public.stockbook_sales s
    join public.stockbook_products p on p.id = s.product_id
    where s.company_id = v_company
      and (p_all_time or (
        s.sold_at >= coalesce(p_from, '-infinity'::timestamptz)
        and s.sold_at <= coalesce(p_to, 'infinity'::timestamptz)
      ))
      and (p_category is null or p_category = '' or lower(p.category) = lower(p_category))
    order by s.sold_at desc, s.id desc
    limit v_page_size offset (v_page - 1) * v_page_size
  ) x;

  select coalesce(jsonb_agg(jsonb_build_object(
      'category', category,
      'revenue', revenue,
      'profit', profit,
      'units', units
    ) order by revenue desc), '[]'::jsonb)
  into v_breakdown
  from (
    select min(p.category) as category,
           coalesce(sum(s.total_price), 0)::bigint as revenue,
           coalesce(sum(s.profit), 0)::bigint as profit,
           coalesce(sum(s.quantity), 0)::bigint as units
    from public.stockbook_sales s
    join public.stockbook_products p on p.id = s.product_id
    where s.company_id = v_company
      and (p_all_time or (
        s.sold_at >= coalesce(p_from, '-infinity'::timestamptz)
        and s.sold_at <= coalesce(p_to, 'infinity'::timestamptz)
      ))
      and (p_category is null or p_category = '' or lower(p.category) = lower(p_category))
    group by lower(p.category)
    order by revenue desc
    limit 6
  ) x;

  return jsonb_build_object(
    'rows', v_rows,
    'summary', v_summary,
    'categoryBreakdown', v_breakdown,
    'page', v_page,
    'totalPages', v_pages,
    'pageSize', v_page_size,
    'categories', public.stockbook_categories()
  );
end;
$$;

-- ------------------------------------------------------------------- lookup

create or replace function public.stockbook_product(p_product_id bigint)
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_product jsonb;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  select jsonb_build_object(
      'id', id,
      'name', name,
      'sku', sku,
      'category', category,
      'quantity', quantity,
      'purchasePrice', purchase_price,
      'sellPrice', sell_price,
      'lowStockThreshold', low_stock_threshold
    )
  into v_product
  from public.stockbook_products
  where id = p_product_id and company_id = v_company;

  return v_product;
end;
$$;

create or replace function public.stockbook_product_options()
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  return coalesce((
    select jsonb_agg(jsonb_build_object(
        'id', id,
        'name', name,
        'sku', sku,
        'category', category,
        'quantity', quantity,
        'purchasePrice', purchase_price,
        'sellPrice', sell_price,
        'lowStockThreshold', low_stock_threshold
      ) order by lower(name) asc)
    from public.stockbook_products
    where company_id = v_company
  ), '[]'::jsonb);
end;
$$;

-- ---------------------------------------------------------------- item writes

create or replace function public.stockbook_create_item(
  p_name text,
  p_sku text,
  p_category text,
  p_purchase_price bigint,
  p_sell_price bigint,
  p_low_stock_threshold bigint
)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  insert into public.stockbook_products (
    company_id, name, sku, category, purchase_price, sell_price,
    quantity, low_stock_threshold, created_at, updated_at
  ) values (
    v_company, p_name, nullif(p_sku, ''), p_category, p_purchase_price, p_sell_price,
    0, p_low_stock_threshold, now(), now()
  );

  return jsonb_build_object('ok', true);
exception
  when unique_violation then
    return jsonb_build_object(
      'ok', false, 'field', 'sku',
      'error', 'An item with this SKU already exists.'
    );
end;
$$;

create or replace function public.stockbook_update_item(
  p_product_id bigint,
  p_name text,
  p_sku text,
  p_category text,
  p_purchase_price bigint,
  p_sell_price bigint,
  p_low_stock_threshold bigint
)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  update public.stockbook_products
  set name = p_name,
      sku = nullif(p_sku, ''),
      category = p_category,
      purchase_price = p_purchase_price,
      sell_price = p_sell_price,
      low_stock_threshold = p_low_stock_threshold,
      updated_at = now()
  where id = p_product_id and company_id = v_company;

  if not found then
    return jsonb_build_object('ok', false, 'error', 'Item not found.');
  end if;

  return jsonb_build_object('ok', true);
exception
  when unique_violation then
    return jsonb_build_object(
      'ok', false, 'field', 'sku',
      'error', 'An item with this SKU already exists.'
    );
end;
$$;

-- ------------------------------------------------------------ purchase writes

create or replace function public.stockbook_create_purchase(
  p_product_id bigint,
  p_quantity bigint,
  p_unit_cost bigint,
  p_purchased_at timestamptz,
  p_note text
)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_name text;
  v_purchase_id bigint;
  v_total bigint;
  v_when timestamptz := coalesce(p_purchased_at, now());
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  select name into v_name
  from public.stockbook_products
  where id = p_product_id and company_id = v_company
  for update;

  if not found then
    return jsonb_build_object(
      'ok', false, 'field', 'productId',
      'error', 'Select an item from your catalog.'
    );
  end if;

  v_total := p_quantity * p_unit_cost;

  insert into public.stockbook_purchases (
    company_id, product_id, quantity, unit_cost, total_cost,
    purchased_at, note, created_at
  ) values (
    v_company, p_product_id, p_quantity, p_unit_cost, v_total,
    v_when, nullif(p_note, ''), now()
  )
  returning id into v_purchase_id;

  update public.stockbook_products
  set quantity = quantity + p_quantity,
      purchase_price = p_unit_cost,
      updated_at = now()
  where id = p_product_id and company_id = v_company;

  return jsonb_build_object(
    'ok', true, 'purchaseId', v_purchase_id, 'productName', v_name
  );
end;
$$;

create or replace function public.stockbook_delete_purchase(p_purchase_id bigint)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_product_id bigint;
  v_name text;
  v_quantity bigint;
  v_stock bigint;
  v_latest_cost bigint;
  v_plural text;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  select pu.product_id, p.name, pu.quantity, p.quantity
  into v_product_id, v_name, v_quantity, v_stock
  from public.stockbook_purchases pu
  join public.stockbook_products p on p.id = pu.product_id
  where pu.id = p_purchase_id
    and pu.company_id = v_company
    and p.company_id = v_company
  for update of p;

  if not found then
    return jsonb_build_object('ok', false, 'error', 'That purchase record no longer exists.');
  end if;

  if v_stock < v_quantity then
    v_plural := case when v_quantity = 1 then '' else 's' end;
    return jsonb_build_object(
      'ok', false,
      'error', format(
        'Cannot delete this purchase: it added %s unit%s of %s, but only %s remain in stock. Sell or adjust stock before deleting it.',
        v_quantity, v_plural, v_name, v_stock
      )
    );
  end if;

  delete from public.stockbook_purchases where id = p_purchase_id and company_id = v_company;

  select unit_cost into v_latest_cost
  from public.stockbook_purchases
  where product_id = v_product_id and company_id = v_company
  order by purchased_at desc, id desc
  limit 1;

  if found then
    update public.stockbook_products
    set quantity = quantity - v_quantity,
        purchase_price = v_latest_cost,
        updated_at = now()
    where id = v_product_id and company_id = v_company;
  else
    update public.stockbook_products
    set quantity = quantity - v_quantity,
        updated_at = now()
    where id = v_product_id and company_id = v_company;
  end if;

  return jsonb_build_object('ok', true, 'productName', v_name);
end;
$$;

-- ---------------------------------------------------------------- sale writes

create or replace function public.stockbook_create_sale(
  p_product_id bigint,
  p_quantity bigint,
  p_unit_price bigint,
  p_payment_mode text,
  p_note text
)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_name text;
  v_stock bigint;
  v_cost bigint;
  v_sale_id bigint;
  v_total_price bigint;
  v_total_cost bigint;
  v_plural text;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  if p_payment_mode not in ('cash', 'upi') then
    return jsonb_build_object(
      'ok', false, 'field', 'paymentMode',
      'error', 'Select a valid payment mode.'
    );
  end if;

  select name, quantity, purchase_price
  into v_name, v_stock, v_cost
  from public.stockbook_products
  where id = p_product_id and company_id = v_company
  for update;

  if not found then
    return jsonb_build_object('ok', false, 'error', 'That product is no longer in your inventory.');
  end if;

  if v_stock < p_quantity then
    v_plural := case when v_stock = 1 then '' else 's' end;
    return jsonb_build_object(
      'ok', false, 'field', 'quantity',
      'error', format('Only %s unit%s of %s in stock.', v_stock, v_plural, v_name)
    );
  end if;

  v_total_price := p_quantity * p_unit_price;
  v_total_cost := p_quantity * v_cost;

  insert into public.stockbook_sales (
    company_id, product_id, quantity, unit_price, total_price,
    unit_cost, total_cost, profit, payment_mode, sold_at, note, created_at
  ) values (
    v_company, p_product_id, p_quantity, p_unit_price, v_total_price,
    v_cost, v_total_cost, v_total_price - v_total_cost,
    p_payment_mode, now(), nullif(p_note, ''), now()
  )
  returning id into v_sale_id;

  update public.stockbook_products
  set quantity = quantity - p_quantity, updated_at = now()
  where id = p_product_id and company_id = v_company;

  return jsonb_build_object('ok', true, 'saleId', v_sale_id, 'productName', v_name);
end;
$$;

create or replace function public.stockbook_delete_sale(p_sale_id bigint)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_product_id bigint;
  v_name text;
  v_quantity bigint;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  select s.product_id, p.name, s.quantity
  into v_product_id, v_name, v_quantity
  from public.stockbook_sales s
  join public.stockbook_products p on p.id = s.product_id
  where s.id = p_sale_id
    and s.company_id = v_company
    and p.company_id = v_company
  for update of p;

  if not found then
    return jsonb_build_object('ok', false, 'error', 'That sale record no longer exists.');
  end if;

  delete from public.stockbook_sales where id = p_sale_id and company_id = v_company;

  update public.stockbook_products
  set quantity = quantity + v_quantity, updated_at = now()
  where id = v_product_id and company_id = v_company;

  return jsonb_build_object('ok', true, 'productName', v_name);
end;
$$;

-- --------------------------------------------------------------- backup/restore

create or replace function public.stockbook_export()
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_row public.stockbook_companies%rowtype;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  select * into v_row from public.stockbook_companies where id = v_company;

  return jsonb_build_object(
    'format', 'stockbook-backup',
    'version', 1,
    'exportedAt', now(),
    'company', jsonb_build_object('name', v_row.name, 'email', v_row.email),
    'products', coalesce((
      select jsonb_agg(jsonb_build_object(
          'id', p.id,
          'name', p.name,
          'sku', p.sku,
          'category', p.category,
          'purchasePrice', p.purchase_price,
          'sellPrice', p.sell_price,
          'quantity', p.quantity,
          'lowStockThreshold', p.low_stock_threshold,
          'createdAt', p.created_at,
          'updatedAt', p.updated_at
        ) order by p.id)
      from public.stockbook_products p
      where p.company_id = v_company
    ), '[]'::jsonb),
    'purchases', coalesce((
      select jsonb_agg(jsonb_build_object(
          'id', pu.id,
          'productId', pu.product_id,
          'quantity', pu.quantity,
          'unitCost', pu.unit_cost,
          'totalCost', pu.total_cost,
          'purchasedAt', pu.purchased_at,
          'note', pu.note,
          'createdAt', pu.created_at
        ) order by pu.id)
      from public.stockbook_purchases pu
      where pu.company_id = v_company
    ), '[]'::jsonb),
    'sales', coalesce((
      select jsonb_agg(jsonb_build_object(
          'id', s.id,
          'productId', s.product_id,
          'quantity', s.quantity,
          'unitPrice', s.unit_price,
          'totalPrice', s.total_price,
          'unitCost', s.unit_cost,
          'totalCost', s.total_cost,
          'profit', s.profit,
          'paymentMode', s.payment_mode,
          'soldAt', s.sold_at,
          'note', s.note,
          'createdAt', s.created_at
        ) order by s.id)
      from public.stockbook_sales s
      where s.company_id = v_company
    ), '[]'::jsonb)
  );
end;
$$;

create or replace function public.stockbook_import(p_data jsonb)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_company bigint := public.stockbook_company_id();
  v_item jsonb;
  v_new_product bigint;
  v_map jsonb := '{}'::jsonb;
  v_products integer := 0;
  v_purchases integer := 0;
  v_sales integer := 0;
begin
  if v_company is null then
    raise exception 'unauthorized';
  end if;

  if p_data is null
     or p_data->>'format' is distinct from 'stockbook-backup'
     or jsonb_typeof(p_data->'products') is distinct from 'array'
     or jsonb_typeof(p_data->'purchases') is distinct from 'array'
     or jsonb_typeof(p_data->'sales') is distinct from 'array' then
    return jsonb_build_object('ok', false, 'error', 'That file is not a Stockbook backup.');
  end if;

  delete from public.stockbook_sales where company_id = v_company;
  delete from public.stockbook_purchases where company_id = v_company;
  delete from public.stockbook_products where company_id = v_company;

  for v_item in select * from jsonb_array_elements(p_data->'products')
  loop
    insert into public.stockbook_products (
      company_id, name, sku, category, purchase_price, sell_price,
      quantity, low_stock_threshold, created_at, updated_at
    ) values (
      v_company,
      coalesce(v_item->>'name', 'Untitled item'),
      nullif(v_item->>'sku', ''),
      coalesce(v_item->>'category', 'Uncategorised'),
      coalesce((v_item->>'purchasePrice')::bigint, 0),
      coalesce((v_item->>'sellPrice')::bigint, 0),
      coalesce((v_item->>'quantity')::bigint, 0),
      coalesce((v_item->>'lowStockThreshold')::bigint, 5),
      coalesce(nullif(v_item->>'createdAt', '')::timestamptz, now()),
      coalesce(nullif(v_item->>'updatedAt', '')::timestamptz, now())
    )
    returning id into v_new_product;

    v_map := jsonb_set(v_map, array[v_item->>'id'], to_jsonb(v_new_product));
    v_products := v_products + 1;
  end loop;

  for v_item in select * from jsonb_array_elements(p_data->'purchases')
  loop
    v_new_product := nullif(v_map->>(v_item->>'productId'), '')::bigint;
    if v_new_product is null then
      continue;
    end if;

    insert into public.stockbook_purchases (
      company_id, product_id, quantity, unit_cost, total_cost,
      purchased_at, note, created_at
    ) values (
      v_company,
      v_new_product,
      greatest(coalesce((v_item->>'quantity')::bigint, 1), 1),
      coalesce((v_item->>'unitCost')::bigint, 0),
      coalesce((v_item->>'totalCost')::bigint, 0),
      coalesce(nullif(v_item->>'purchasedAt', '')::timestamptz, now()),
      nullif(v_item->>'note', ''),
      coalesce(nullif(v_item->>'createdAt', '')::timestamptz, now())
    );
    v_purchases := v_purchases + 1;
  end loop;

  for v_item in select * from jsonb_array_elements(p_data->'sales')
  loop
    v_new_product := nullif(v_map->>(v_item->>'productId'), '')::bigint;
    if v_new_product is null then
      continue;
    end if;

    insert into public.stockbook_sales (
      company_id, product_id, quantity, unit_price, total_price,
      unit_cost, total_cost, profit, payment_mode, sold_at, note, created_at
    ) values (
      v_company,
      v_new_product,
      greatest(coalesce((v_item->>'quantity')::bigint, 1), 1),
      coalesce((v_item->>'unitPrice')::bigint, 0),
      coalesce((v_item->>'totalPrice')::bigint, 0),
      coalesce((v_item->>'unitCost')::bigint, 0),
      coalesce((v_item->>'totalCost')::bigint, 0),
      coalesce((v_item->>'profit')::bigint, 0),
      case when v_item->>'paymentMode' = 'upi' then 'upi' else 'cash' end,
      coalesce(nullif(v_item->>'soldAt', '')::timestamptz, now()),
      nullif(v_item->>'note', ''),
      coalesce(nullif(v_item->>'createdAt', '')::timestamptz, now())
    );
    v_sales := v_sales + 1;
  end loop;

  return jsonb_build_object(
    'ok', true,
    'products', v_products,
    'purchases', v_purchases,
    'sales', v_sales
  );
end;
$$;

-- ---------------------------------------------------------------- RLS + grants

alter table public.stockbook_companies enable row level security;
alter table public.stockbook_products enable row level security;
alter table public.stockbook_purchases enable row level security;
alter table public.stockbook_sales enable row level security;

drop policy if exists stockbook_companies_own on public.stockbook_companies;
create policy stockbook_companies_own on public.stockbook_companies
  for select to authenticated
  using (user_id = auth.uid());

drop policy if exists stockbook_products_own on public.stockbook_products;
create policy stockbook_products_own on public.stockbook_products
  for all to authenticated
  using (company_id = public.stockbook_company_id())
  with check (company_id = public.stockbook_company_id());

drop policy if exists stockbook_purchases_own on public.stockbook_purchases;
create policy stockbook_purchases_own on public.stockbook_purchases
  for all to authenticated
  using (company_id = public.stockbook_company_id())
  with check (company_id = public.stockbook_company_id());

drop policy if exists stockbook_sales_own on public.stockbook_sales;
create policy stockbook_sales_own on public.stockbook_sales
  for all to authenticated
  using (company_id = public.stockbook_company_id())
  with check (company_id = public.stockbook_company_id());

grant usage on schema public to authenticated;
grant select on public.stockbook_companies to authenticated;
grant select, insert, update, delete on public.stockbook_products to authenticated;
grant select, insert, update, delete on public.stockbook_purchases to authenticated;
grant select, insert, update, delete on public.stockbook_sales to authenticated;
grant usage, select on all sequences in schema public to authenticated;

grant execute on function public.stockbook_company_id() to authenticated;
grant execute on function public.stockbook_escape_like(text) to authenticated;
grant execute on function public.stockbook_bootstrap_company(text) to authenticated;
grant execute on function public.stockbook_categories() to authenticated;
grant execute on function public.stockbook_overview(timestamptz, timestamptz, boolean, text, integer) to authenticated;
grant execute on function public.stockbook_items(text, text, integer, integer) to authenticated;
grant execute on function public.stockbook_inventory(text, text, boolean, integer, integer) to authenticated;
grant execute on function public.stockbook_purchases(timestamptz, timestamptz, boolean, text, integer, integer) to authenticated;
grant execute on function public.stockbook_sales(timestamptz, timestamptz, boolean, text, integer, integer) to authenticated;
grant execute on function public.stockbook_product(bigint) to authenticated;
grant execute on function public.stockbook_product_options() to authenticated;
grant execute on function public.stockbook_create_item(text, text, text, bigint, bigint, bigint) to authenticated;
grant execute on function public.stockbook_update_item(bigint, text, text, text, bigint, bigint, bigint) to authenticated;
grant execute on function public.stockbook_create_purchase(bigint, bigint, bigint, timestamptz, text) to authenticated;
grant execute on function public.stockbook_delete_purchase(bigint) to authenticated;
grant execute on function public.stockbook_create_sale(bigint, bigint, bigint, text, text) to authenticated;
grant execute on function public.stockbook_delete_sale(bigint) to authenticated;
grant execute on function public.stockbook_export() to authenticated;
grant execute on function public.stockbook_import(jsonb) to authenticated;

notify pgrst, 'reload schema';

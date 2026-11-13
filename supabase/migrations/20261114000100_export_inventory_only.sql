-- Export files contain inventory data only (items, purchases, sales).
-- Account fields such as company name, email address, and password hashes
-- must never be written into an export, so replace the original function
-- that also returned a "company" block.

create or replace function public.stockbook_export()
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

  return jsonb_build_object(
    'format', 'stockbook-backup',
    'version', 1,
    'exportedAt', now(),
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

package main

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	emailPattern  = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	moneyPattern  = regexp.MustCompile(`^\d{1,7}(\.\d{1,2})?$`)
	digitsPattern = regexp.MustCompile(`^\d{1,7}$`)
	idPattern     = regexp.MustCompile(`^\d{1,12}$`)
)

type fieldErrors map[string][]string

func (f fieldErrors) add(field, message string) {
	f[field] = append(f[field], message)
}

func (f fieldErrors) result() map[string][]string {
	if len(f) == 0 {
		return nil
	}
	return f
}

type validRegister struct {
	CompanyName string
	Email       string
	Password    string
}

func validateRegister(in RegisterInput) (validRegister, map[string][]string) {
	errs := fieldErrors{}
	out := validRegister{}

	companyName := strings.TrimSpace(in.CompanyName)
	switch {
	case companyName == "":
		errs.add("companyName", "Company name is required.")
	case len([]rune(companyName)) > 80:
		errs.add("companyName", "Company name must be 80 characters or fewer.")
	}
	out.CompanyName = companyName

	email := strings.ToLower(strings.TrimSpace(in.Email))
	switch {
	case email == "":
		errs.add("email", "Email is required.")
	case len(email) > 254:
		errs.add("email", "Email is too long.")
	case !emailPattern.MatchString(email):
		errs.add("email", "Enter a valid email address.")
	}
	out.Email = email

	password := in.Password
	if len(password) < 8 {
		errs.add("password", "Password must be at least 8 characters.")
	}
	if len(password) > 72 {
		errs.add("password", "Password must be 72 characters or fewer.")
	}
	if !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") {
		errs.add("password", "Password must include a letter.")
	}
	if !strings.ContainsAny(password, "0123456789") {
		errs.add("password", "Password must include a number.")
	}
	out.Password = password

	if in.ConfirmPassword == "" {
		errs.add("confirmPassword", "Confirm your password.")
	} else if password != in.ConfirmPassword {
		errs.add("confirmPassword", "Passwords do not match.")
	}

	// Match the sign-up form: collect at most three issues per field.
	for field, messages := range errs {
		if len(messages) > 3 {
			errs[field] = messages[:3]
		}
	}
	return out, errs.result()
}

type validLogin struct {
	Email    string
	Password string
}

func validateLogin(in LoginInput) (validLogin, map[string][]string) {
	errs := fieldErrors{}
	out := validLogin{}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	switch {
	case email == "":
		errs.add("email", "Email is required.")
	case len(email) > 254:
		errs.add("email", "Email is too long.")
	case !emailPattern.MatchString(email):
		errs.add("email", "Enter a valid email address.")
	}
	out.Email = email

	switch {
	case in.Password == "":
		errs.add("password", "Password is required.")
	case len(in.Password) > 200:
		errs.add("password", "Password is too long.")
	}
	out.Password = in.Password

	return out, errs.result()
}

type validItem struct {
	Name              string
	SKU               *string
	Category          string
	LowStockThreshold int64
	PurchasePrice     int64
	SellPrice         int64
}

func validateItem(in ItemInput, requireID bool) (validItem, map[string][]string) {
	errs := fieldErrors{}
	out := validItem{}

	if requireID {
		if _, ok := parseRecordID(strconv.FormatInt(in.ProductID, 10)); !ok {
			errs.add("productId", "Invalid selection.")
		}
	}

	name := strings.TrimSpace(in.Name)
	switch {
	case name == "":
		errs.add("name", "Item name is required.")
	case len([]rune(name)) > 100:
		errs.add("name", "Item name must be 100 characters or fewer.")
	}
	out.Name = name

	sku := strings.TrimSpace(in.SKU)
	if len([]rune(sku)) > 40 {
		errs.add("sku", "SKU must be 40 characters or fewer.")
	} else if sku != "" {
		out.SKU = &sku
	}

	category := strings.Join(strings.Fields(in.Category), " ")
	switch {
	case category == "":
		errs.add("category", "Category is required.")
	case len([]rune(category)) > 50:
		errs.add("category", "Category must be 50 characters or fewer.")
	}
	out.Category = category

	thresholdRaw := strings.TrimSpace(in.LowStockThreshold)
	if !digitsPattern.MatchString(thresholdRaw) {
		errs.add("lowStockThreshold", "Low-stock alert must be a whole number.")
	} else {
		threshold, _ := strconv.ParseInt(thresholdRaw, 10, 64)
		if threshold > 1_000_000 {
			errs.add("lowStockThreshold", "Alert must be 1,000,000 or less.")
		} else {
			out.LowStockThreshold = threshold
		}
	}

	if cents, message := parseMoney(in.PurchasePrice); message != "" {
		errs.add("purchasePrice", message)
	} else {
		out.PurchasePrice = cents
	}
	if cents, message := parseMoney(in.SellPrice); message != "" {
		errs.add("sellPrice", message)
	} else {
		out.SellPrice = cents
	}

	return out, errs.result()
}

type validPurchase struct {
	ProductID   int64
	Quantity    int64
	UnitCost    int64
	PurchasedAt string
	Note        *string
}

func validatePurchase(in PurchaseInput) (validPurchase, map[string][]string) {
	errs := fieldErrors{}
	out := validPurchase{}

	productID, ok := parseRecordID(in.ProductID)
	if !ok {
		errs.add("productId", "Invalid selection.")
	}
	out.ProductID = productID

	quantity, message := parsePositiveQuantity(in.Quantity)
	if message != "" {
		errs.add("quantity", message)
	}
	out.Quantity = quantity

	unitCost, message := parseMoney(in.UnitCost)
	if message != "" {
		errs.add("unitCost", message)
	}
	out.UnitCost = unitCost

	purchasedAt := strings.TrimSpace(in.PurchasedAt)
	if !isISODate(purchasedAt) {
		errs.add("purchasedAt", "Enter a valid purchase date.")
	}
	out.PurchasedAt = purchasedAt
	out.Note = parseNote(in.Note, errs)

	return out, errs.result()
}

type validSale struct {
	ProductID   int64
	Quantity    int64
	UnitPrice   int64
	PaymentMode string
	Note        *string
}

func validateSale(in SaleInput) (validSale, map[string][]string) {
	errs := fieldErrors{}
	out := validSale{}

	productID, ok := parseRecordID(in.ProductID)
	if !ok {
		errs.add("productId", "Invalid selection.")
	}
	out.ProductID = productID

	quantity, message := parsePositiveQuantity(in.Quantity)
	if message != "" {
		errs.add("quantity", message)
	}
	out.Quantity = quantity

	unitPrice, message := parseMoney(in.UnitPrice)
	if message != "" {
		errs.add("unitPrice", message)
	}
	out.UnitPrice = unitPrice

	if in.PaymentMode != "cash" && in.PaymentMode != "upi" {
		errs.add("paymentMode", "Select a valid payment mode.")
	}
	out.PaymentMode = in.PaymentMode
	out.Note = parseNote(in.Note, errs)

	return out, errs.result()
}

func parseMoney(raw string) (int64, string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, "Enter an amount."
	}
	if !moneyPattern.MatchString(value) {
		return 0, "Enter a valid amount up to 9,999,999.99."
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, "Enter a valid amount up to 9,999,999.99."
	}
	cents := int64(math.Round(number * 100))
	if cents < 0 || cents > 1_000_000_000 {
		return 0, "Amount must be 9,999,999.99 or less."
	}
	return cents, ""
}

func parsePositiveQuantity(raw string) (int64, string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, "Enter a quantity."
	}
	if !digitsPattern.MatchString(value) {
		return 0, "Quantity must be a whole number."
	}
	quantity, err := strconv.ParseInt(value, 10, 64)
	if err != nil || quantity <= 0 {
		return 0, "Quantity must be greater than zero."
	}
	if quantity > 1_000_000 {
		return 0, "Quantity must be 1,000,000 or less."
	}
	return quantity, ""
}

func parseRecordID(raw string) (int64, bool) {
	value := strings.TrimSpace(raw)
	if !idPattern.MatchString(value) {
		return 0, false
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func parseNote(raw string, errs fieldErrors) *string {
	note := strings.TrimSpace(raw)
	if len([]rune(note)) > 240 {
		errs.add("note", "Note must be 240 characters or fewer.")
		return nil
	}
	if note == "" {
		return nil
	}
	return &note
}

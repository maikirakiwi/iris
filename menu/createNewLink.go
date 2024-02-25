package menu

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/stripe/stripe-go/v76"

	DB "iris/v2/database"
	"iris/v2/models"
	stripeapi "iris/v2/stripe-api"
)

func CreateNewLink() {
	settings := &models.Settings{}
	db_res := DB.Conn.First(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error)
		return
	}
	stripe.Key = settings.ApiKey
	params := new(stripe.CheckoutSessionParams)

	prompt := promptui.Prompt{
		Label: "Set Session Link ID (to be used /pay/your-link-id)",
	}
	linkID, _ := prompt.Run()

	prompt = promptui.Prompt{
		Label: "Set Session Link Nickname",
	}
	nick, _ := prompt.Run()

	prompt = promptui.Prompt{
		Label: "Set Max Uses",
	}
	maxuses, _ := prompt.Run()
	maxusesInt, err := strconv.Atoi(maxuses)
	if err != nil {
		println("Error: %v\n", err)
		return
	}

	params.AllowPromotionCodes = stripe.Bool(allowCoupons())

	prompt = promptui.Prompt{
		Label: "Ask for shipping address? (y/n)",
	}
	shippingAddress, _ := prompt.Run()
	if err != nil {
		println("Error: %v\n", err)
		return
	}
	if shippingAddress == "y" {
		params.ShippingAddressCollection = &stripe.CheckoutSessionShippingAddressCollectionParams{
			AllowedCountries: stripe.StringSlice(DB.GetShippingCountries()),
		}
	}

	li, trackedItems := addItems(*settings)

	for _, trackedItem := range trackedItems {
		tracking := &models.Inventory{}
		DB.Conn.Where(&models.Inventory{Product: trackedItem}).First(&tracking)
		tracking.SessionLinkIDs = append(tracking.SessionLinkIDs, linkID)
		DB.Conn.Save(&tracking)
	}

	params.LineItems = li
	params.Mode = stripe.String("payment")
	params.SuccessURL = stripe.String("http://" + settings.Domain + "/success")

	// Optional
	customFields := allowCustomFields()
	if customFields != nil {
		params.CustomFields = customFields
	}
	// Optional
	invoices := allowInvoices()
	if invoices != nil {
		params.InvoiceCreation = invoices
	}

	err = DB.Conn.Create(&models.SessionLink{
		LinkID:               linkID,
		Active:               true,
		Nickname:             nick,
		MaxUses:              maxusesInt,
		Params:               models.SessionParams{CheckoutSessionParams: *params},
		TrackingInventoryIDs: trackedItems,
	}).Error
	if err != nil {
		println("Error while adding link to database: " + err.Error())
		return
	}

	println("Created Link: /pay/" + linkID)

}

func askQtyandPrice() (int64, int64, error) {
	prompt := promptui.Prompt{
		Label: "Qty and Price in cents, separated by space. (e.g. 3x $1.99 = 3 199)",
	}
	input, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(-1)
		}
		println("Error: %v\n", err)
		return 0, 0, err
	}
	split := strings.Split(input, " ")
	if len(split) != 2 {
		println("Error: Invalid input")
		return 0, 0, err
	}
	qty, err := strconv.ParseInt(split[0], 10, 64)
	if err != nil {
		println("Error: Invalid input")
		return 0, 0, err
	}
	perPrice, err := strconv.ParseInt(split[1], 10, 64)
	if err != nil {
		println("Error: Invalid input")
		return 0, 0, err
	}
	return qty, perPrice, nil
}

// Line items and tracked inventory IDs
func addItems(settings models.Settings) ([]*stripe.CheckoutSessionLineItemParams, []string) {
	items := []*stripe.CheckoutSessionLineItemParams{}
	trackedItems := []string{}
	rawProducts, err := stripeapi.GetAllProduct(true)
	if err != nil {
		println("Error while reading products: " + err.Error())
		return nil, nil
	}
	readableProducts := []string{"Finish Adding"}
	for _, product := range rawProducts {
		inv := &models.Inventory{}

		err := DB.Conn.Find(&inv, "product = ?", product.ID).Error
		if err != nil {
			println("Error while reading inventory database: " + err.Error())
			return nil, nil
		}
		if inv.Quantity == 0 {
			readableProducts = append(readableProducts, product.Name+" ("+product.ID+")")
		} else {
			readableProducts = append(readableProducts, fmt.Sprintf("%s (%d left) (%s)", product.Name, inv.Quantity, product.ID))
		}
	}

	for {
		productSelection := promptui.Select{
			Label: "Select a product to add",
			Items: readableProducts,
		}
		index, res, _ := productSelection.Run()
		if res == "Finish Adding" {
			break
		}
		qty, perPrice, err := askQtyandPrice()
		if err != nil {
			continue
		}
		itemID, err := stripeapi.NewPriceIfNotExist(settings.DefaultCurrency, perPrice, rawProducts[index-1].ID)
		if err != nil {
			println("Error while adding item: " + err.Error())
			continue
		}

		prompt := promptui.Prompt{
			Label: "Allow customer adjustable quantity for this item? (y/n)",
		}
		input, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			println("Error: %v\n", err)
			continue
		}

		if input == "y" {
			prompt = promptui.Prompt{
				Label: "Set min and max adjustable qty separated by space. e.g. min 1 and max 5 is 1 5",
			}
			input, err = prompt.Run()
			if err != nil {
				if err == promptui.ErrInterrupt {
					os.Exit(-1)
				}
				println("Error: %v\n", err)
				continue
			}
			adjQty := strings.Split(input, " ")
			if len(adjQty) != 2 {
				println("Error: Invalid input")
				continue
			}
			adjMin, err := strconv.ParseInt(adjQty[0], 10, 64)
			if err != nil {
				println("Error: Invalid input")
				continue
			}
			adjMax, err := strconv.ParseInt(adjQty[1], 10, 64)
			if err != nil {
				println("Error: Invalid input")
				continue
			}

			items = append(items, &stripe.CheckoutSessionLineItemParams{
				Price:    stripe.String(itemID),
				Quantity: stripe.Int64(qty),
				AdjustableQuantity: &stripe.CheckoutSessionLineItemAdjustableQuantityParams{
					Enabled: stripe.Bool(true),
					Minimum: stripe.Int64(adjMin),
					Maximum: stripe.Int64(adjMax),
				},
			})
		} else {
			items = append(items, &stripe.CheckoutSessionLineItemParams{
				Price:    stripe.String(itemID),
				Quantity: stripe.Int64(qty),
			})
		}
		if strings.Contains(readableProducts[index], " left) (") {
			trackedItems = append(trackedItems, rawProducts[index-1].ID)
		}
	}

	return items, trackedItems
}

func allowCoupons() bool {
	prompt := promptui.Prompt{
		Label: "Allow coupons and promotion codes? (y/n)",
	}
	allowCoupon, err := prompt.Run()
	if err != nil {
		println("Error: %v\n", err)
		return false
	}
	if allowCoupon == "y" {
		return true
	}
	return false
}

func allowCustomFields() []*stripe.CheckoutSessionCustomFieldParams {
	prompt := promptui.Prompt{
		Label: "Number of Custom Fields to add (up to 2), leave blank to skip",
	}
	input, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(-1)
		}
		println("Error: %v\n", err)
		return nil
	}
	inputInt, err := strconv.Atoi(input)
	if err != nil && input != "" {
		println("Error: %v\n", err)
		return nil
	}
	selectedCustomFields := []*stripe.CheckoutSessionCustomFieldParams{}
	allCustomFields := []models.CustomFields{}
	db_res := DB.Conn.Find(&allCustomFields)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return nil
	}

	if len(allCustomFields) == 0 && (input != "" || inputInt > 0) {
		println("No custom fields found, please create some first.")
	} else if input != "" || inputInt > 0 {
		readableFields := []string{}
		for _, field := range allCustomFields {
			readableFields = append(readableFields, field.Label)
		}
		for i := 0; i < inputInt; i++ {
			prompt := promptui.Select{
				Label: fmt.Sprintf("Select (%d/%d) Custom Field to add", i+1, inputInt),
				Items: readableFields,
			}
			index, _, err := prompt.Run()
			if err != nil {
				if err == promptui.ErrInterrupt {
					os.Exit(-1)
				}
				println("Prompt failed %v\n", err)
				panic(err)
			}
			selectedCustomFields = append(selectedCustomFields, allCustomFields[index].ToStripe())
		}

		return selectedCustomFields
	}

	return nil
}

func allowInvoices() *stripe.CheckoutSessionInvoiceCreationParams {
	prompt := promptui.Prompt{
		Label: "Generate a post-purchase Invoice? (y/n)",
	}
	invoiceEnabled, err := prompt.Run()
	if err != nil {
		println("Error: %v\n", err)
		return nil
	}
	allInvoiceTemplates := []models.InvoicePDF{}
	db_res := DB.Conn.Find(&allInvoiceTemplates)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return nil
	}

	if invoiceEnabled == "y" && len(allInvoiceTemplates) == 0 {
		println("No invoice templates found, please create some first.")
	} else if invoiceEnabled == "y" {
		readableInvoices := []string{}
		for _, inv := range allInvoiceTemplates {
			readableInvoices = append(readableInvoices, inv.Nickname)
		}

		prompt := promptui.Select{
			Label: "Select an invoice template to attach",
			Items: readableInvoices,
		}
		index, _, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			println("Prompt failed %v\n", err)
			return nil
		}

		res := new(stripe.CheckoutSessionInvoiceCreationParams)
		res.Enabled = stripe.Bool(true)
		res.InvoiceData = &stripe.CheckoutSessionInvoiceCreationInvoiceDataParams{}
		if allInvoiceTemplates[index].TaxID != "" {
			res.InvoiceData.AccountTaxIDs = []*string{
				&allInvoiceTemplates[index].TaxID,
			}
		}
		if allInvoiceTemplates[index].CustomFieldName != "" {
			res.InvoiceData.CustomFields = []*stripe.CheckoutSessionInvoiceCreationInvoiceDataCustomFieldParams{
				{
					Name:  stripe.String(allInvoiceTemplates[index].CustomFieldName),
					Value: stripe.String(allInvoiceTemplates[index].CustomFieldValue),
				},
			}
		}
		if allInvoiceTemplates[index].Description != "" {
			res.InvoiceData.Description = stripe.String(allInvoiceTemplates[index].Description)
		}
		if allInvoiceTemplates[index].Footer != "" {
			res.InvoiceData.Footer = stripe.String(allInvoiceTemplates[index].Footer)
		}

		return res
	}

	return nil
}

package menu

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"

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
	params := new(stripe.PaymentLinkParams)

	prompt := promptui.Prompt{
		Label: "Set Payment Link Nickname",
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

	prompt = promptui.Prompt{
		Label: "Allow coupons and promotion codes? (y/n)",
	}
	allowCoupon, _ := prompt.Run()
	if err != nil {
		println("Error: %v\n", err)
		return
	}
	if allowCoupon == "y" {
		params.AllowPromotionCodes = stripe.Bool(true)
	} else {
		params.AllowPromotionCodes = stripe.Bool(false)
	}

	prompt = promptui.Prompt{
		Label: "Ask for shipping address? (y/n)",
	}
	shippingAddress, _ := prompt.Run()
	if err != nil {
		println("Error: %v\n", err)
		return
	}
	if shippingAddress == "y" {
		params.ShippingAddressCollection = &stripe.PaymentLinkShippingAddressCollectionParams{
			AllowedCountries: stripe.StringSlice([]string{
				"AC", "AD", "AE", "AF", "AG", "AI", "AL", "AM", "AO", "AQ", "AR", "AT", "AU", "AW", "AX", "AZ", "BA", "BB", "BD", "BE", "BF", "BG", "BH", "BI", "BJ", "BL", "BM", "BN", "BO", "BQ", "BR", "BS", "BT", "BV", "BW", "BY", "BZ", "CA", "CD", "CF", "CG", "CH", "CI", "CK", "CL", "CM", "CN", "CO", "CR", "CV", "CW", "CY", "CZ", "DE", "DJ", "DK", "DM", "DO", "DZ", "EC", "EE", "EG", "EH", "ER", "ES", "ET", "FI", "FJ", "FK", "FO", "FR", "GA", "GB", "GD", "GE", "GF", "GG", "GH", "GI", "GL", "GM", "GN", "GP", "GQ", "GR", "GS", "GT", "GU", "GW", "GY", "HK", "HN", "HR", "HT", "HU", "ID", "IE", "IL", "IM", "IN", "IO", "IQ", "IS", "IT", "JE", "JM", "JO", "JP", "KE", "KG", "KH", "KI", "KM", "KN", "KR", "KW", "KY", "KZ", "LA", "LB", "LC", "LI", "LK", "LR", "LS", "LT", "LU", "LV", "LY", "MA", "MC", "MD", "ME", "MF", "MG", "MK", "ML", "MM", "MN", "MO", "MQ", "MR", "MS", "MT", "MU", "MV", "MW", "MX", "MY", "MZ", "NA", "NC", "NE", "NG", "NI", "NL", "NO", "NP", "NR", "NU", "NZ", "OM", "PA", "PE", "PF", "PG", "PH", "PK", "PL", "PM", "PN", "PR", "PS", "PT", "PY", "QA", "RE", "RO", "RS", "RU", "RW", "SA", "SB", "SC", "SE", "SG", "SH", "SI", "SJ", "SK", "SL", "SM", "SN", "SO", "SR", "SS", "ST", "SV", "SX", "SZ", "TA", "TC", "TD", "TF", "TG", "TH", "TJ", "TK", "TL", "TM", "TN", "TO", "TR", "TT", "TV", "TW", "TZ", "UA", "UG", "US", "UY", "UZ", "VA", "VC", "VE", "VG", "VN", "VU", "WF", "WS", "XK", "YE", "YT", "ZA", "ZM", "ZW", "ZZ",
			}),
		}
	}

	items := []*stripe.PaymentLinkLineItemParams{}

	for {
		fmt.Println("Product ID, qty and Price in cents (e.g. $1.99 = 199), separated by space.")
		prompt := promptui.Prompt{
			Label: "Add Line Item or leave blank to finish",
		}
		input, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			println("Error: %v\n", err)
			return
		}
		if input == "" {
			break
		}
		split := strings.Split(input, " ")
		if len(split) != 3 {
			println("Error: Invalid input")
			continue
		}
		qty, err := strconv.ParseInt(split[1], 10, 64)
		if err != nil {
			println("Error: Invalid input")
			continue
		}
		perPrice, err := strconv.ParseInt(split[2], 10, 64)
		if err != nil {
			println("Error: Invalid input")
			continue
		}

		itemID, err := stripeapi.NewPriceIfNotExist(settings.DefaultCurrency, perPrice, split[0])
		if err != nil {
			println("Error while adding item: " + err.Error())
			continue
		}

		prompt = promptui.Prompt{
			Label: "Allow customer adjustable quantity for this item? (y/n)",
		}
		input, err = prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			println("Error: %v\n", err)
			return
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
				return
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

			items = append(items, &stripe.PaymentLinkLineItemParams{
				Price:    stripe.String(itemID),
				Quantity: stripe.Int64(qty),
				AdjustableQuantity: &stripe.PaymentLinkLineItemAdjustableQuantityParams{
					Enabled: stripe.Bool(true),
					Minimum: stripe.Int64(adjMin),
					Maximum: stripe.Int64(adjMax),
				},
			})
		} else {
			items = append(items, &stripe.PaymentLinkLineItemParams{
				Price:    stripe.String(itemID),
				Quantity: stripe.Int64(qty),
			})
		}

	}

	prompt = promptui.Prompt{
		Label: "Number of Custom Fields to add (up to 2), leave blank to skip",
	}
	input, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(-1)
		}
		println("Error: %v\n", err)
		return
	}
	inputInt, err := strconv.Atoi(input)
	if err != nil && input != "" {
		println("Error: %v\n", err)
		return
	}
	selectedCustomFields := []*stripe.PaymentLinkCustomFieldParams{}
	allCustomFields := []models.CustomFields{}
	db_res = DB.Conn.Find(&allCustomFields)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return
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
				return
			}
			selectedCustomFields = append(selectedCustomFields, allCustomFields[index].ToStripe())
		}

		params.CustomFields = selectedCustomFields
	}

	// Stripe hates empty custom fields
	params.LineItems = items
	paymentConfirmation := DB.GetSettings().PaymentConfirmationMessage
	if paymentConfirmation != "" {
		params.AfterCompletion = &stripe.PaymentLinkAfterCompletionParams{
			Type: stripe.String("hosted_confirmation"),
			HostedConfirmation: &stripe.PaymentLinkAfterCompletionHostedConfirmationParams{
				CustomMessage: stripe.String(paymentConfirmation),
			},
		}
	}

	link, err := paymentlink.New(params)
	if err != nil {
		println("Error while creating link: " + err.Error())
		return
	}

	err = DB.Conn.Create(&models.PaymentLink{
		Active:   true,
		Nickname: nick,
		LinkID:   link.ID,
		URL:      link.URL,
		Used:     0,
		MaxUses:  maxusesInt,
	}).Error
	if err != nil {
		println("Error while adding link to database: " + err.Error())
		return
	}

	println("Link: " + link.URL)

}

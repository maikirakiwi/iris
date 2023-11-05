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
	var allowCouponBool bool
	if allowCoupon == "y" {
		allowCouponBool = true
	} else {
		allowCouponBool = false
	}

	items := []*stripe.PaymentLinkLineItemParams{}

	for {
		prompt := promptui.Prompt{
			Label: "Product ID, qty and Price in cents (e.g. $1.99 = 199), separated by space. Leave blank to finish.",
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
	}

	// Stripe hates empty custom fields
	var params *stripe.PaymentLinkParams
	params.LineItems = items
	params.AllowPromotionCodes = stripe.Bool(allowCouponBool)
	if len(selectedCustomFields) != 0 {
		params.CustomFields = selectedCustomFields
	}
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

package menu

import (
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

	items := []*stripe.PaymentLinkLineItemParams{}

	for {
		prompt := promptui.Prompt{
			Label: "Product ID, qty and Price in USD cents, separated by space. Leave blank to finish.",
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

		item, err := stripeapi.NewPriceIfNotExist(settings.DefaultCurrency, perPrice, split[0])
		if err != nil {
			println("Error while adding item: " + err.Error())
			continue
		}

		items = append(items, &stripe.PaymentLinkLineItemParams{
			Price:    stripe.String(item.PriceID),
			Quantity: stripe.Int64(qty),
		})
	}

	params := &stripe.PaymentLinkParams{
		LineItems: items,
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

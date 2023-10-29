package menu

import (
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"
	"github.com/stripe/stripe-go/v76/price"

	DB "stripe-handler/v2/database"
	"stripe-handler/v2/models"
)

func Main() {
	prompt := promptui.Select{
		Label: "Menu",
		Items: []string{"Create New Link", "Modify Existing Link",
			"Change Webhook Endpoint Secret", "Change Stripe API Key"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(-1)
		}
		println("Prompt failed %v\n", err)
		return
	}

	switch result {
	case "Change Stripe API Key":
		ChangeApiKey()
	case "Change Webhook Endpoint Secret":
		ChangeEndpointSecret()
	case "Create New Link":
		CreateNewLink()
	case "Modify Existing Link":
		ModifyExistingLink()
	}
}

func ModifyExistingLink() {
	settings := &models.Settings{}
	db_res := DB.Conn.First(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error)
		return
	}
	stripe.Key = settings.ApiKey

	allLinks := []models.PaymentLink{}
	db_res = DB.Conn.Find(&allLinks)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error)
		return
	}

	readableLinks := []string{}
	for _, link := range allLinks {
		if link.Active {
			readableLinks = append(readableLinks, "[Active] "+link.Nickname+" ("+strconv.Itoa(link.Used)+"/"+strconv.Itoa(link.MaxUses)+")")
		} else {
			readableLinks = append(readableLinks, "[Inactive] "+link.Nickname+" ("+strconv.Itoa(link.Used)+"/"+strconv.Itoa(link.MaxUses)+")")
		}
	}

	prompt := promptui.Select{
		Label: "Select Link to Modify",
		Items: readableLinks,
	}
	selection, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(-1)
		}
		println("Prompt failed %v\n", err)
		return
	}

	prompt = promptui.Select{
		Label: "-> " + readableLinks[selection],
		Items: []string{"Change Nickname", "Change Max Uses", "Activate/Deactivate Link", "Delete Link"},
	}
	_, choice, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(-1)
		}
		println("Prompt failed %v\n", err)
		return
	}
	switch choice {
	case "Change Nickname":
		prompt := promptui.Prompt{
			Label: "Change Payment Link Nickname",
		}
		nick, _ := prompt.Run()
		allLinks[selection].Nickname = nick
		DB.Conn.Save(&allLinks[selection])

	case "Change Max Uses":
		prompt := promptui.Prompt{
			Label: "Change Max Uses",
		}
		maxuses, _ := prompt.Run()
		maxusesInt, err := strconv.Atoi(maxuses)
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			println("Error: %v\n", err)
			return
		}
		allLinks[selection].MaxUses = maxusesInt
		DB.Conn.Save(&allLinks[selection])
	case "Activate/Deactivate Link":
		var q promptui.Prompt
		if allLinks[selection].Active {
			q = promptui.Prompt{
				Label:     "Deactivate Link?",
				IsConfirm: true,
			}
		} else {
			q = promptui.Prompt{
				Label:     "Activate Link?",
				IsConfirm: true,
			}
			allLinks[selection].Used = 0
		}
		result, err := q.Run()
		if result == "n" || err != nil {
			return
		}
		pl, err := paymentlink.Update(
			allLinks[selection].LinkID,
			&stripe.PaymentLinkParams{
				Active: stripe.Bool(!allLinks[selection].Active),
			},
		)
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			println("Error while changing link on Stripe: " + err.Error())
			return
		}
		allLinks[selection].Active = pl.Active
		DB.Conn.Save(&allLinks[selection])
	case "Delete Link":
		prompt := promptui.Prompt{
			Label:     "Delete Link Locally & Deactivate on Stripe?",
			IsConfirm: true,
		}
		result, _ := prompt.Run()
		if result == "y" {
			_, err := paymentlink.Update(
				allLinks[selection].LinkID,
				&stripe.PaymentLinkParams{
					Active: stripe.Bool(false),
				},
			)
			if err != nil {
				println("Error while changing link on Stripe: " + err.Error())
				return
			}
			DB.Conn.Delete(&allLinks[selection])
		}
	}
	Main()
}

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

		item, err := price.New(&stripe.PriceParams{
			Currency:   stripe.String(string(stripe.CurrencyUSD)),
			UnitAmount: stripe.Int64(perPrice),
			Product:    stripe.String(split[0]),
		})
		if err != nil {
			println("Error while adding item: " + err.Error())
			continue
		}

		items = append(items, &stripe.PaymentLinkLineItemParams{
			Price:    stripe.String(item.ID),
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

func ChangeEndpointSecret() {
	prompt := promptui.Prompt{
		Label: "Set Stripe Webhook Endpoint Secret",
	}
	result, _ := prompt.Run()

	settings := models.Settings{}
	db_res := DB.Conn.FirstOrCreate(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error)
		return
	}
	settings.WebhookEndpointSecret = result
	DB.Conn.Save(&settings)

	Main()
}

func ChangeApiKey() {
	prompt := promptui.Prompt{
		Label: "Set Stripe API Key",
	}

	result, _ := prompt.Run()

	settings := models.Settings{}
	db_res := DB.Conn.FirstOrCreate(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error)
		return
	}
	settings.ApiKey = result
	DB.Conn.Save(&settings)

	Main()
}

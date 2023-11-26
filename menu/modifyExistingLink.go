package menu

import (
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/stripe/stripe-go/v76"

	DB "iris/v2/database"
	"iris/v2/models"
	stripeapi "iris/v2/stripe-api"
)

func ModifyExistingLink() {
	settings := &models.Settings{}
	db_res := DB.Conn.First(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error)
		return
	}
	stripe.Key = settings.ApiKey

	allLinks := []models.SessionLink{}
	db_res = DB.Conn.Find(&allLinks)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error)
		return
	}

	readableLinks := []string{}
	for _, link := range allLinks {
		if link.Active {
			readableLinks = append(readableLinks, "[Active] "+link.Nickname+" ("+strconv.Itoa(link.Used)+"/"+strconv.Itoa(link.MaxUses)+")")
			continue
		}
		readableLinks = append(readableLinks, "[Inactive] "+link.Nickname+" ("+strconv.Itoa(link.Used)+"/"+strconv.Itoa(link.MaxUses)+")")
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
		Items: []string{"Change Nickname", "Change List Item Price", "Change Max Uses", "Activate/Deactivate Link", "Delete Link"},
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
			Label: "Change Session Link Nickname",
		}
		nick, _ := prompt.Run()
		allLinks[selection].Nickname = nick
		DB.Conn.Save(&allLinks[selection])
	case "Change List Item Price":
		prompt := promptui.Select{
			Label: "Change Item Price",
			Items: stripeapi.GetProductsInLink(allLinks[selection].LinkID),
		}
		itemSelection, _, _ := prompt.Run()
		pricePrompt := promptui.Prompt{
			Label: "Price in cents. (e.g. $1.99 = 199)",
		}
		input, err := pricePrompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			println("Error: %v\n", err)
			return
		}
		split := strings.Split(input, " ")
		if len(split) != 2 {
			println("Error: Invalid input")
			return
		}
		priceInt, err := strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			println("Error: Invalid input")
			return
		}
		if err != nil {
			println("Error: %v\n", err)
			return
		}
		price := &models.Price{}
		DB.Conn.Where(&models.Price{PriceID: *allLinks[selection].Params.LineItems[itemSelection].Price}).First(price)
		newPrice, err := stripeapi.NewPriceIfNotExist(settings.DefaultCurrency, priceInt, price.Product)
		if err != nil {
			println("Error: %v\n", err)
			return
		}
		*allLinks[selection].Params.LineItems[itemSelection].Price = newPrice
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
				Label: "Deactivate Link? (y/n)",
			}
		} else {
			q = promptui.Prompt{
				Label: "Activate Link? (y/n)",
			}
			allLinks[selection].Used = 0
		}
		result, err := q.Run()
		if result == "n" || err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			return
		}
		allLinks[selection].Active = !allLinks[selection].Active
		DB.Conn.Save(&allLinks[selection])
	case "Delete Link":
		prompt := promptui.Prompt{
			Label: "Delete Link Permanently? (y/n)",
		}
		result, _ := prompt.Run()
		if result == "y" {
			if err != nil {
				println("Error while changing link on Stripe: " + err.Error())
				return
			}
			DB.Conn.Delete(&allLinks[selection])
		}
	}
}

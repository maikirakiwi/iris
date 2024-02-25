package menu

import (
	"os"
	"slices"
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
		Items: []string{"Change Nickname", "Change List Item Price", "Change Max Uses", "Disable Shipping Countries", "Activate/Deactivate Link", "Delete Link"},
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
		priceInt, err := strconv.ParseInt(input, 10, 64)
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
	case "Disable Shipping Countries":
		prompt := promptui.Prompt{
			Label: "Disable Shipping Countries via list of ISO alpha-2 codes, Seperated by Space.",
		}
		input, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			println("Error: %v\n", err)
		}
		split := strings.Split(input, " ")
		defaultCountries := []string{
			"AC", "AD", "AE", "AF", "AG", "AI", "AL", "AM", "AO", "AQ", "AR", "AT", "AU", "AW", "AX", "AZ", "BA", "BB", "BD", "BE", "BF", "BG", "BH", "BI", "BJ", "BL", "BM", "BN", "BO", "BQ", "BR", "BS", "BT", "BV", "BW", "BY", "BZ", "CA", "CD", "CF", "CG", "CH", "CI", "CK", "CL", "CM", "CN", "CO", "CR", "CV", "CW", "CY", "CZ", "DE", "DJ", "DK", "DM", "DO", "DZ", "EC", "EE", "EG", "EH", "ER", "ES", "ET", "FI", "FJ", "FK", "FO", "FR", "GA", "GB", "GD", "GE", "GF", "GG", "GH", "GI", "GL", "GM", "GN", "GP", "GQ", "GR", "GS", "GT", "GU", "GW", "GY", "HK", "HN", "HR", "HT", "HU", "ID", "IE", "IL", "IM", "IN", "IO", "IQ", "IS", "IT", "JE", "JM", "JO", "JP", "KE", "KG", "KH", "KI", "KM", "KN", "KR", "KW", "KY", "KZ", "LA", "LB", "LC", "LI", "LK", "LR", "LS", "LT", "LU", "LV", "LY", "MA", "MC", "MD", "ME", "MF", "MG", "MK", "ML", "MM", "MN", "MO", "MQ", "MR", "MS", "MT", "MU", "MV", "MW", "MX", "MY", "MZ", "NA", "NC", "NE", "NG", "NI", "NL", "NO", "NP", "NR", "NU", "NZ", "OM", "PA", "PE", "PF", "PG", "PH", "PK", "PL", "PM", "PN", "PR", "PS", "PT", "PY", "QA", "RE", "RO", "RS", "RU", "RW", "SA", "SB", "SC", "SE", "SG", "SH", "SI", "SJ", "SK", "SL", "SM", "SN", "SO", "SR", "SS", "ST", "SV", "SX", "SZ", "TA", "TC", "TD", "TF", "TG", "TH", "TJ", "TK", "TL", "TM", "TN", "TO", "TR", "TT", "TV", "TW", "TZ", "UA", "UG", "US", "UY", "UZ", "VA", "VC", "VE", "VG", "VN", "VU", "WF", "WS", "XK", "YE", "YT", "ZA", "ZM", "ZW", "ZZ",
		}
		filteredCountries := stripe.StringSlice([]string{})
		for _, country := range defaultCountries {
			if slices.Contains(split, country) {
				continue
			}
			filteredCountries = append(filteredCountries, stripe.String(country))
		}

		allLinks[selection].Params.ShippingAddressCollection.AllowedCountries = filteredCountries
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

package menu

import (
	"os"
	"strconv"

	"github.com/manifoldco/promptui"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"

	DB "iris/v2/database"
	"iris/v2/models"
)

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
		pl, err := paymentlink.Update(
			allLinks[selection].LinkID,
			&stripe.PaymentLinkParams{
				Active: stripe.Bool(!allLinks[selection].Active),
			},
		)
		if err != nil {
			println("Error while changing link on Stripe: " + err.Error())
			return
		}
		allLinks[selection].Active = pl.Active
		DB.Conn.Save(&allLinks[selection])
	case "Delete Link":
		prompt := promptui.Prompt{
			Label: "Delete Link Locally & Deactivate on Stripe? (y/n)",
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
}
package menu

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"

	DB "iris/v2/database"
	"iris/v2/models"
)

func ChangeEndpointSecret() {
	prompt := promptui.Prompt{
		Label: "Set Stripe Webhook Endpoint Secret",
	}
	result, _ := prompt.Run()

	settings := models.Settings{}
	db_res := DB.Conn.FirstOrCreate(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return
	}
	settings.WebhookEndpointSecret = result
	DB.Conn.Save(&settings)
}

func ChangeApiKey() {
	prompt := promptui.Prompt{
		Label: "Set Stripe API Key",
	}

	result, _ := prompt.Run()

	settings := models.Settings{}
	db_res := DB.Conn.FirstOrCreate(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return
	}
	settings.ApiKey = result
	DB.Conn.Save(&settings)
}

func ChangeDefaultCurrency() {
	prompt := promptui.Prompt{
		Label: "Change Default Charge Currency, List of supported currencies: https://stripe.com/docs/currencies",
	}

	result, _ := prompt.Run()

	settings := models.Settings{}
	db_res := DB.Conn.FirstOrCreate(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return
	}
	settings.DefaultCurrency = strings.ToLower(result)
	DB.Conn.Save(&settings)
}

func ChangePaymentConfirmationMsg() {
	prompt := promptui.Prompt{
		Label: fmt.Sprintf("Change Payment Confirmation Message (Currently: %s)", DB.GetSettings().PaymentConfirmationMessage),
	}

	result, _ := prompt.Run()

	settings := models.Settings{}
	db_res := DB.Conn.FirstOrCreate(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return
	}
	settings.PaymentConfirmationMessage = result
	DB.Conn.Save(&settings)
}

func ChangeDomain() {
	prompt := promptui.Prompt{
		Label: fmt.Sprintf("Change Machine Domain (Currently: %s)", DB.GetSettings().Domain),
	}

	result, _ := prompt.Run()

	settings := models.Settings{}
	db_res := DB.Conn.FirstOrCreate(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return
	}
	settings.Domain = result
	DB.Conn.Save(&settings)
}

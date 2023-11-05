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

func AddCustomFields() {
	prompt := promptui.Prompt{
		Label: "Set Custom Field Label",
	}
	label, err := prompt.Run()
	if err != nil {
		println("Error: %v\n", err.Error())
		return
	}

	promptSelect := promptui.Select{
		Label: "Set Custom Field Type",
		Items: []string{"Text", "Numeric", "Dropdown"},
	}

	var dropdowns []models.DropdownOption
	_, fieldType, err := promptSelect.Run()
	if err != nil {
		println("Error: %v\n", err.Error())
		return
	}
	fieldType = strings.ToLower(fieldType)
	if fieldType == "dropdown" {
		for i := 0; i < 200; i++ {
			prompt := promptui.Prompt{
				Label: fmt.Sprintf("Add a dropdown option (%d/200), or leave empty to finish", i+1),
			}
			dropdown, err := prompt.Run()
			if err != nil {
				println("Error: %v\n", err.Error())
				return
			}
			if dropdown == "" {
				break
			}
			dropdowns = append(dropdowns, models.DropdownOption{
				Label:    dropdown,
				RecValue: strings.ReplaceAll(dropdown+label+fmt.Sprintf("%d", i), " ", ""),
			})
		}
	}
	customField := models.CustomFields{
		Label:    label,
		Type:     fieldType,
		Dropdown: dropdowns,
	}
	DB.Conn.Create(&customField)
}

func DeleteCustomFields() {
	allFields := []models.CustomFields{}
	db_res := DB.Conn.Find(&allFields)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return
	}

	readableFields := []string{}
	for _, field := range allFields {
		readableFields = append(readableFields, field.Label)
	}
	prompt := promptui.Select{
		Label: "Delete Custom Fields",
		Items: readableFields,
	}
	index, _, err := prompt.Run()
	if err != nil {
		println("Error: %v\n", err.Error())
		return
	}

	DB.Conn.Delete(&allFields[index])
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
		Label: fmt.Sprintf("Change Payment Confirmation Message, or set to 0 to disable (Currently: %s)", DB.GetSettings().PaymentConfirmationMessage),
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

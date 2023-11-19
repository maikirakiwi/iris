package menu

import (
	"github.com/manifoldco/promptui"

	DB "iris/v2/database"
	"iris/v2/models"
)

func CreateInvoiceTemplate() {
	invoice := &models.InvoicePDF{}
	var err error

	prompt := promptui.Prompt{
		Label: "Set Invoice Template Nickname",
	}
	invoice.Nickname, err = prompt.Run()
	if err != nil {
		println("Error: %v\n", err.Error())
		return
	}

	prompt = promptui.Prompt{
		Label: "The account tax ID associated with the invoice (Optional)",
	}
	invoice.TaxID, err = prompt.Run()
	if err != nil {
		println("Error: %v\n", err.Error())
		return
	}

	prompt = promptui.Prompt{
		Label: "Custom Field Name on the invoice (Optional, 30 chars max)",
	}
	customfieldname, err := prompt.Run()
	if err != nil {
		println("Error: %v\n", err.Error())
		return
	}

	if customfieldname != "" {
		prompt = promptui.Prompt{
			Label: "Custom Field Value on the invoice (Optional, 30 chars max)",
		}
		customfieldvalue, err := prompt.Run()
		if err != nil {
			println("Error: %v\n", err.Error())
			return
		}
		invoice.CustomFieldName = customfieldname
		invoice.CustomFieldValue = customfieldvalue
	}

	prompt = promptui.Prompt{
		Label: "Description to be displayed on the invoice (Optional)",
	}
	invoice.Description, err = prompt.Run()
	if err != nil {
		println("Error: %v\n", err.Error())
		return
	}

	prompt = promptui.Prompt{
		Label: "Footer to be displayed on the invoice (Optional)",
	}
	invoice.Footer, err = prompt.Run()
	if err != nil {
		println("Error: %v\n", err.Error())
		return
	}

	DB.Conn.Create(&invoice)
}

func DeleteInvoiceTemplate() {
	allInvoices := []models.InvoicePDF{}
	db_res := DB.Conn.Find(&allInvoices)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return
	}

	readableFields := []string{}
	for _, inv := range allInvoices {
		readableFields = append(readableFields, inv.Nickname)
	}

	prompt := promptui.Select{
		Label: "Delete Invoice Template",
		Items: readableFields,
	}
	index, _, err := prompt.Run()
	if err != nil {
		println("Error: %v\n", err.Error())
		return
	}

	DB.Conn.Delete(&allInvoices[index])
}

package menu

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"

	DB "iris/v2/database"
	"iris/v2/models"
)

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

package menu

import (
	"os"

	"github.com/manifoldco/promptui"
)

func Entry() {
	sessionLinksOpt := []string{"Create New Link", "Modify Existing Link"}
	customFieldOpt := []string{"Create Custom Field", "Delete Custom Field"}
	stripeSettingsOpt := []string{"Change Stripe API Key", "Change Webhook Endpoint Secret", "Change Default Currency", "Change Custom Payment Confirmation Message", "Change Machine Domain", "Change Default Shipping Countries"}
	invoicePDFOpt := []string{"Create Invoice Template", "Delete Invoice Template"}
	inventoryOpt := []string{"Add Inventory to Track", "Change Inventory Quantity", "Delete Existing Inventory"}
	prompt := promptui.Select{
		Label: "Iris Menu (Ctrl+C to exit at any time)",
		Items: []string{
			"Manage Session Links",
			"Manage Invoice Templates",
			"Manage Inventory",
			"Manage Custom Fields",
			"Manage Stripe Settings",
		},
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
	case "Manage Session Links":
		prompt = promptui.Select{
			Label: "Session Links",
			Items: sessionLinksOpt,
		}
	case "Manage Invoice Templates":
		prompt = promptui.Select{
			Label: "Invoice Templates",
			Items: invoicePDFOpt,
		}
	case "Manage Custom Fields":
		prompt = promptui.Select{
			Label: "Custom Fields",
			Items: customFieldOpt,
		}
	case "Manage Stripe Settings":
		prompt = promptui.Select{
			Label: "Stripe Settings",
			Items: stripeSettingsOpt,
		}
	case "Manage Inventory":
		println("Current Inventory:")
		for _, inv := range ListInventory() {
			println(inv)
		}
		prompt = promptui.Select{
			Label: "Select Inventory Action",
			Items: inventoryOpt,
		}
	}

	_, result, err = prompt.Run()
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
		Entry()
	case "Change Webhook Endpoint Secret":
		ChangeEndpointSecret()
		Entry()
	case "Create New Link":
		CreateNewLink()
		Entry()
	case "Modify Existing Link":
		ModifyExistingLink()
		Entry()
	case "Create Custom Field":
		AddCustomFields()
		Entry()
	case "Delete Custom Field":
		DeleteCustomFields()
		Entry()
	case "Change Default Currency":
		ChangeDefaultCurrency()
		Entry()
	case "Change Custom Payment Confirmation Message":
		ChangePaymentConfirmationMsg()
		Entry()
	case "Create Invoice Template":
		CreateInvoiceTemplate()
		Entry()
	case "Delete Invoice Template":
		DeleteInvoiceTemplate()
		Entry()
	case "Add Inventory to Track":
		AddTrackingInventory()
		Entry()
	case "Change Inventory Quantity":
		ChangeInventoryQuantity()
		Entry()
	case "Delete Existing Inventory":
		RemoveTrackingInventory()
		Entry()
	case "Change Machine Domain":
		ChangeDomain()
		Entry()
	case "Change Default Shipping Countries":
		ChangeDefaultShippingCountries()
		Entry()
	}
}

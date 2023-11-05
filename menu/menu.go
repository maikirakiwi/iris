package menu

import (
	"os"

	"github.com/manifoldco/promptui"
)

func Entry() {
	paymentLinksOpt := []string{"Create New Link", "Modify Existing Link"}
	customFieldOpt := []string{"Create Custom Field", "Delete Custom Field"}
	stripeSettingsOpt := []string{"Change Stripe API Key", "Change Webhook Endpoint Secret", "Change Default Currency", "Change Custom Payment Confirmation Message"}
	prompt := promptui.Select{
		Label: "Iris Menu (Ctrl+C to exit at any time)",
		Items: []string{
			"Manage Payment Links",
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
	case "Manage Payment Links":
		prompt = promptui.Select{
			Label: "Payment Links",
			Items: paymentLinksOpt,
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
	}
}

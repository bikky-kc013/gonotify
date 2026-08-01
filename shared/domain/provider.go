package domain

type Provider string

const (
	ProviderSendGrid Provider = "SENDGRID"
	ProviderSES      Provider = "SES"
	ProviderTwilio   Provider = "TWILIO"
	ProviderMSG91    Provider = "MSG91"
	ProviderFCM      Provider = "FCM"
	ProviderAPNS     Provider = "APNS"
)

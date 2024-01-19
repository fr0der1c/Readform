package main

var otpRequestChan chan string
var otpChan chan string

func init() {
	otpRequestChan = make(chan string)
	otpChan = make(chan string) // this have to be a string, not int64, to be compatible with otp codes starts with zero
}

func RequireOTP(agentDisplayName string) string {
	otpRequestChan <- agentDisplayName
	return <-otpChan
}

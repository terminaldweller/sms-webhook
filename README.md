# sms-webhook

A simple sms webhook with plain authentication to get the SMS you receive on your android phone on IRC.</br>
I have a made blogpost about it [here](https://blog.terminaldweller.com/posts/how_to_get_your_sms_on_irc).</br>

For IRC bot supports SASL plain authentication.</br>
The webhook endpoint itself supports HTTP basic authentication.</br>
The webhook has [pocketbase](https://github.com/pocketbase/pocketbase) integrated so you can use that to create new users.</br>

Last but not least, you will need a forwarding agent that actually sends the SMS you get on your android device to the webhook endpoint.</br>
Currently [this](https://github.com/bogkonstantin/android_income_sms_gateway_webhook) is what I'm using to forwars my SMS to the webhook. Also make sure the app settings on android are changed accordingly because the forwarder needs to run in the background so make sure android does not battery-optimize it out of existence.</br>

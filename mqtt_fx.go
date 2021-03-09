package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

//mosquitto憑證

const rootPEM = `
-----BEGIN CERTIFICATE-----
-----END CERTIFICATE-----
`

//啟動MQTT
func RunMqtt(broker string, userName string, passWord string) MQTT.Client {

	opts := MQTT.NewClientOptions().AddBroker(broker)

	// initial
	if broker[0:3] == "ssl" { // it is TLS connection
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM([]byte(rootPEM))
		if !ok {
			// panic("Failed to parse root certificate")
			log.Println("Failed to parse root certificate")
		}
		clientKeyPair, err := tls.LoadX509KeyPair("client-crt.pem", "client-key.pem")
		if err == nil {
			tlsconf := &tls.Config{RootCAs: roots, ClientAuth: tls.NoClientCert, ClientCAs: roots, InsecureSkipVerify: true, Certificates: []tls.Certificate{clientKeyPair}}
			opts.SetTLSConfig(tlsconf)
		} else {
			log.Println(`Failed to parse Client Certificates`)
		}
	}

	opts.SetClientID(``)
	opts.SetCleanSession(true)
	opts.SetMaxReconnectInterval(1 * time.Second)
	opts.SetAutoReconnect(true) //自動重連
	opts.SetMessageChannelDepth(300)
	opts.SetConnectionLostHandler(ConnLostHandler)
	opts.SetOnConnectHandler(OnConnectHandler)
	opts.SetKeepAlive(60 * time.Second)
	if userName != "" {
		opts.SetUsername(userName)
		opts.SetPassword(passWord)
	}
	mqc := MQTT.NewClient(opts)
	for {
		if token := mqc.Connect(); token.Wait() && token.Error() != nil {
			// panic(token.Error())
			log.Println("Connect Mqtt failed:", token.Error())
			time.Sleep(5 * time.Second)
			mqc = MQTT.NewClient(opts)
		} else {
			log.Printf("Connected to broker %s\n", broker)
			break
		}
	}
	log.Println("MQTT connect to", broker)
	return mqc
}

func MqttPublish(mqc MQTT.Client, topic string, retain bool, msg string) error {
	log.Println("Into mqttPublish")
	if topic == "" {
		log.Println("Topic is null, msg:", msg)
		return errors.New("Empty TOPIC")
	} else {
		if mqc != nil {
			//Publish(topic string, qos byte, retained bool, payload interface{})
			if tkn := mqc.Publish(topic, 1, retain, msg); tkn.Wait() && tkn.Error() != nil {
				log.Println("Publish error:", tkn.Error())
				return tkn.Error()
			}
			optr := mqc.OptionsReader()
			log.Printf("Broker: %s\n", optr.Servers()[0].String())
			log.Printf("Topic: [%s] Pub: %s\n", topic, msg)
		} else {
			log.Println("MQTT client is nil, pub error:", topic)
		}
	}
	return nil
}

func MqttSubscribe(mqc MQTT.Client, topic string) {
	if mqc != nil {
		if token := mqc.Subscribe(topic, 1, MqttReceiveCallback); token.Wait() && token.Error() != nil {
			// panic(token.Error())
			log.Println("MQTT sub error:", token.Error())
		} else {
			// all fine
			optr := mqc.OptionsReader()
			log.Printf("Broker: %s\n", optr.Servers()[0].String())
			log.Println("MQTT sub ok:", topic)
		}
	} else {
		log.Println("MQTT client is nil, sub error:", topic)
	}
}

// call-back when message arrived
func MqttReceiveCallback(client MQTT.Client, msg MQTT.Message) {
	mpl := msg.Payload()
	mtp := msg.Topic()
	//optr := client.OptionsReader()
	//log.Printf("Broker: %s\n", optr.Servers()[0].String())
	//log.Printf("Topic: [%s] Msg: %s\n", mtp, mpl)
	MqChan <- MqMsg{Topic: string(mtp), Payload: mpl}
	//cmd := string(mpl)
	//tok := strings.Split(cmd, ` `)
}

//捕獲連接丟失
func ConnLostHandler(client MQTT.Client, err error) {
	log.Printf("Connection lost, reason: %v\n", err)
	//Perform additional action...
	// TODO: 重連時間是否有限制
}

//OnConnectHandler call-back function for 1st connection or re-connections
func OnConnectHandler(client MQTT.Client) {
	log.Println("Into OnConnectHandler")
	optsr := client.OptionsReader()
	if len(SUBTOPIC) != 0 {
		log.Println("Resubscribe Topic!")
		MqttSubscribe(client, SUBTOPIC)
		MqttPublish(client, PUBTOPIC, false, optsr.ClientID()+` online`)
	}
}

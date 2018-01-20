package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

var slackEndpoint = os.Getenv("endpoint")
var slackToken = os.Getenv("token")
var expiration = os.Getenv("expiration")

var sess = session.Must(session.NewSession())
var client = iam.New(sess)

type slackResponse struct {
	Ok    bool
	Error string
}

func main() {
	lambda.Start(handler)
}

func handler() {
	users := listUsers()
	log.Println("number of users:", len(users))
	var agedKeys []*iam.AccessKeyMetadata
	age, err := strconv.Atoi(expiration)
	if err != nil {
		log.Println("error converting expiration:", err)
	}
	for _, user := range users {
		agedKeys = append(agedKeys, getAgedKeys(user.UserName, time.Duration(age))...)
	}
	log.Println("number of aged keys:", len(agedKeys))
	if len(agedKeys) == 0 {
		return
	}
	for _, key := range agedKeys {
		slackNotification(key, time.Duration(age))
	}
}

func listUsers() (users []*iam.User) {
	fn := func(output *iam.ListUsersOutput, lastPage bool) bool {
		if len(output.Users) != 0 {
			for _, user := range output.Users {
				if *user.Path != "/service/" {
					users = append(users, user)
				}
			}
		}
		if *output.IsTruncated {
			return !lastPage
		}
		return lastPage
	}
	err := client.ListUsersPages(&iam.ListUsersInput{}, fn)
	if err != nil {
		log.Println(err)
	}
	return
}

func getAgedKeys(user *string, age time.Duration) (keys []*iam.AccessKeyMetadata) {
	fn := func(output *iam.ListAccessKeysOutput, lastPage bool) bool {
		if len(output.AccessKeyMetadata) != 0 {
			for _, data := range output.AccessKeyMetadata {
				if time.Since(*data.CreateDate) > (age+30)*24*time.Hour {
					_, err := client.DeleteAccessKey(&iam.DeleteAccessKeyInput{
						AccessKeyId: data.AccessKeyId,
						UserName:    data.UserName,
					})
					if err != nil {
						log.Println("failed to delete access key %s for user %s", *data.AccessKeyId, *data.UserName)
						log.Println(err)
					}
				}
				if time.Since(*data.CreateDate) > age*24*time.Hour && *data.Status == "Active" {
					_, err := client.UpdateAccessKey(&iam.UpdateAccessKeyInput{
						AccessKeyId: data.AccessKeyId,
						Status:      aws.String("Inactive"),
						UserName:    data.UserName,
					})
					if err != nil {
						log.Println("failed to deactive access key %s for user %s", *data.AccessKeyId, *data.UserName)
						log.Println(err)
					}
					continue
				}
				if time.Since(*data.CreateDate) > (age-7)*24*time.Hour {
					keys = append(keys, data)
				}
			}
		}
		if *output.IsTruncated {
			return !lastPage
		}
		return lastPage
	}
	err := client.ListAccessKeysPages(&iam.ListAccessKeysInput{UserName: user}, fn)
	if err != nil {
		log.Println(err)
	}
	return
}

func slackNotification(key *iam.AccessKeyMetadata, age time.Duration) {
	if *key.UserName != "xiaowei.wang" {
		log.Println("skip slack notification to:", *key.UserName)
		return
	}
	message := []byte(fmt.Sprintf(`
		{
			"channel": "@%s",
			"text": "Your access key %s is expiring and will be disabled at %s. Please generate new access key before expiration. Refer to https://confluence.tingcore-infra.com if any question.",
			"username": "aws-access-key-manager",
			"as_user": false,
			"icon_url": "https://sdk-for-net.amazonwebservices.com/images/AWSLogo128x128.png"
		}`, *key.UserName, *key.AccessKeyId, key.CreateDate.Add(age*24*time.Hour)))
	req, err := http.NewRequest("POST", slackEndpoint, bytes.NewBuffer(message))
	if err != nil {
		log.Println(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf(`Bearer %s`, slackToken))
	req.Header.Set("Content-Type", "application/json")
	log.Println("send slack nofitication to:", *key.UserName)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	log.Println("response status:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println("response body:", string(body))
	var data slackResponse
	_ = json.Unmarshal(body, &data)
	if err != nil {
		log.Println(err)
	}
	if !data.Ok {
		log.Println("failed to send slack notification:", data.Error)
	}
	log.Println("notification delivered to:", *key.UserName)
}

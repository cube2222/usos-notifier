# GCP Setup

## Conventions

I'm using my account name (jakub.wit.martin@gmail.com), use yours if running on your own.

## Service Accounts
* credentials
    * Datastore User
* notifier
    * Datastore User
    
In this document. Whenever a resource is described to be created, it may be followed by a list of service accounts with their respective roles.

## Audit Log

1. Turn on audit logging for cloud kms. Admin read and data read.

## Cloud KMS

1. Create keychain credentials.
    * credentials: Cloud KMS CryptoKey Encrypter/Decrypter
2. Create key credentials in this keychain.

## PubSub

#### Conventions:
* If there is only one publisher to the topic: publisher-topic_name
* If there is more than one publisher: topic_name
* For any subscription: subscriber_name-full_topic_name (includes publisher if applicable)

#### Resources:

1. Create topics:
    * notifications
        * credentials: Pub/Sub Publisher
    * notifier-commands
        * notifier: Pub/Sub Publisher
    * notifier-user_created	
        * notifier: Pub/Sub Publisher
2. Create subscriptions:
    * notifier-notifications
        * notifier: Pub/Sub Subscriber, Pub/Sub Viewer
    * credentials-notifier-user_created
        * credentials: Pub/Sub Subscriber, Pub/Sub Viewer

## Datastore

You need to have Datastore activated. The microservices will create necessary kinds as required.

## Kubernetes Engine

#### Non-defaults:
    * Cluster Version - choose latest
    * Boot disk size - 20GB
    * Network policy - Enabled
    * HTTP load balancing - Disabled
    
#### Compute engine:
    * Turn on https network access to the node, which you will route your DNS to.
    
#### Preliminary cluster setup:
```
    gcloud config set project usos-notifier
    gcloud config set compute/zone europe-west3-b
    gcloud container clusters get-credentials cluster-1 --zone europe-west3-b
    kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
    helm init
    kubectl create clusterrolebinding cluster-admin-binding-jakub.wit.martin@gmail.com --clusterrole=cluster-admin --user=jakub.wit.martin@gmail.com
```

#### Secrets:
* TLS certs for the nginx controller. Standard PEM. I'm using cloudflare generated ones.
    * ```kubectl create secret tls tls-secret --key cert.key --cert cert.crt```
* Credentials service account. Download the json file and call it credentials.json.
    * ```kubectl create secret generic credentials-service-account  --from-file=serviceaccount.json=credentials.json```
* Notifier service account. Download the json file and call it notifier.json.
    * ```kubectl create secret generic notifier-service-account  --from-file=serviceaccount.json=notifier.json```
* Messenger API key. Put the key into your local NOTIFIER_MESSENGER_API_KEY environment variable.
    * On Windows: ```kubectl create secret generic messenger-api --from-literal=messenger-api=$ENV:NOTIFIER_MESSENGER_API_KEY```
    * On Linux: ```kubectl create secret generic messenger-api --from-literal=messenger-api=NOTIFIER_MESSENGER_API_KEY```
* Messenger Verify key. Put the key into your local NOTIFIER_MESSENGER_VERIFY_TOKEN environment variable.
    * On Windows: ```kubectl create secret generic messenger-api --from-literal=messenger-api=$ENV:NOTIFIER_MESSENGER_VERIFY_TOKEN```
    * On Linux: ```kubectl create secret generic messenger-api --from-literal=messenger-api=NOTIFIER_MESSENGER_VERIFY_TOKEN```


#### Infrastructure:
* Nginx controller. This will create a daemon set of nginx instances. All of them will have hostPort 80 and 443, so just route your DNS to one of your nodes.
    * ```helm install --values values.yaml --name nginx-ingress stable/nginx-ingress```
* Ingress. This routes outside traffic to the internal - publicly available - services.
    * ```kubectl apply -f ingress.yaml```

#### Microservices:
* Credentials:
    * ```kubectl apply -f credentials.yaml```
* Notifier:
    * ```kubectl apply -f notifier.yaml```
    
    
#### By the way:
* If cross-compiling windows -> linux you need to ```go get -u golang.org/x/sys/unix```
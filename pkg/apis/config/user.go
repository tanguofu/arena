package config

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	authenticationapi "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getUserName(namespace string, clientConfig clientcmd.ClientConfig, restConfig *rest.Config, clientSet *kubernetes.Clientset, tr *tokenRetriever) (*string, error) {
	tc, err := restConfig.TransportConfig()
	if err != nil {
		return nil, err
	}

	raw, err := clientConfig.RawConfig()
	if err == nil {
		log.Debugf("CurrentContext: %s, raw.Contexts: %v", raw.CurrentContext, raw.Contexts)
		if ctx, ok := raw.Contexts[raw.CurrentContext]; ok {
			userName := ctx.AuthInfo
			return &userName, nil
		}
	} else {
		log.Debugf("get raw config from clientConfig failed, %v", err)
	}

	if tc.HasBasicAuth() {
		userName := fmt.Sprintf("kubecfg:basicauth:%s", restConfig.Username)
		return &userName, nil
	}
	if tc.HasCertAuth() {
		// "userName := kubecfg:certauth:admin"
		userName := fmt.Sprintf("kubecfg:certauth:%s", restConfig.Username)
		return &userName, nil
	}
	var token string
	if tc.HasTokenAuth() {
		if restConfig.BearerTokenFile != "" {
			tokenContent, err := ioutil.ReadFile(restConfig.BearerTokenFile)
			if err != nil {
				return nil, err
			}
			token = string(tokenContent)
		}
		if restConfig.BearerToken != "" {
			token = restConfig.BearerToken
		}
	}
	if token == "" && restConfig.AuthProvider != nil {
		if err := createSubjectRulesReviews(namespace, clientSet); err != nil {
			return nil, err
		}
		token = tr.token
	}
	if token == "" {
		return nil, fmt.Errorf("not found user name for the current context,we don't know how to detect user name")
	}
	return getUserNameByToken(clientSet, token)
}

func getUserNameByToken(kubeclient kubernetes.Interface, token string) (*string, error) {
	result, err := kubeclient.AuthenticationV1().TokenReviews().Create(
		context.TODO(),
		&authenticationapi.TokenReview{
			Spec: authenticationapi.TokenReviewSpec{
				Token: token,
			},
		},
		metav1.CreateOptions{},
	)

	if err != nil {
		return nil, err
	}

	if result.Status.Error != "" {
		return nil, fmt.Errorf(result.Status.Error)
	}

	return &result.Status.User.Username, nil
}

func getAccountFromCert(clientConfig clientcmd.ClientConfig, userName string) (string, string, error) {

	raw, _ := clientConfig.RawConfig()

	for key, user := range raw.AuthInfos {
		log.Debugf("parse raw.AuthInfos key: %s user: %s", key, user.Username)
		if len(user.ClientCertificateData) == 0 {
			continue
		}

		block, _ := pem.Decode(user.ClientCertificateData)
		if block == nil {
			log.Warnf("pem.Decode raw.AuthInfos failed user: %s", user.Username)
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Warnf("x509.ParseCertificate raw.AuthInfos failed user: %s", user.Username)
			continue
		}
		return cert.Subject.Organization[0], cert.Subject.CommonName, nil
	}

	return "", "", errors.NewNotFound(schema.GroupResource{Group: "rbac.authorization.k8s.io", Resource: "ClusterRoleBinding"}, userName)
}
func createSubjectRulesReviews(namespace string, kubeclient kubernetes.Interface) error {
	sar := &authorizationv1.SelfSubjectRulesReview{
		Spec: authorizationv1.SelfSubjectRulesReviewSpec{
			Namespace: namespace,
		},
	}
	_, err := kubeclient.AuthorizationV1().SelfSubjectRulesReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	return err
}

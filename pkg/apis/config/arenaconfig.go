package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"os"

	"github.com/kubeflow/arena/pkg/apis/types"
	"github.com/kubeflow/arena/pkg/util"
	config "github.com/kubeflow/arena/pkg/util/config"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	RecommendedConfigPathEnvVar = "ARENA_CONFIG"
	DefaultArenaConfigPath      = "~/.arena/config"
	GlobalConfigmapName         = "arena-config"
	AdminUserKeyInConfigmap     = "adminUsers"
)

var arenaClient *ArenaConfiger
var errInitArenaClient error

var once sync.Once

// InitArenaConfiger initilize
func InitArenaConfiger(args types.ArenaClientArgs) (*ArenaConfiger, error) {
	once.Do(func() {
		bytes, _ := json.Marshal(args)
		log.Debugf("args: %s", string(bytes))
		arenaClient, errInitArenaClient = newArenaConfiger(args)
	})
	return arenaClient, errInitArenaClient
}

// GetArenaConfiger returns the arena configer,it must be invoked after invoking function InitArenaConfiger(...)
func GetArenaConfiger() *ArenaConfiger {
	if arenaClient == nil {
		err := fmt.Errorf("ArenaClient is not initilized,but you want to get it")
		log.Errorf("Arena Client is not initilized,please use function InitArenaClient(...) to init it")
		panic(err)
	}
	return arenaClient
}

type tokenRetriever struct {
	rountTripper http.RoundTripper
	token        string
}

func (t *tokenRetriever) RoundTrip(req *http.Request) (*http.Response, error) {
	header := req.Header.Get("authorization")
	switch {
	case strings.HasPrefix(header, "Bearer "):
		t.token = strings.ReplaceAll(header, "Bearer ", "")
	}
	return t.rountTripper.RoundTrip(req)
}

type User struct {
	name    string
	id      string
	account string
	group   string
}

func (u User) GetName() string {
	return u.name
}

func (u User) GetId() string {
	return u.id
}

func (u User) GetAccount() string {
	return u.account
}

func (u User) GetGroup() string {
	return u.group
}

type Cluster struct {
	server string
	name   string
}

func (c Cluster) GetServer() string {
	return c.server
}

func (c Cluster) GetName() string {
	return c.name
}

type ArenaConfiger struct {
	restConfig             *rest.Config
	clientConfig           clientcmd.ClientConfig
	clientset              *kubernetes.Clientset
	dynamicClient          dynamic.Interface
	apiExtensionClientset  *extclientset.Clientset
	user                   User
	cluster                Cluster
	adminUsers             []User
	namespace              string
	arenaNamespace         string
	configs                map[string]string
	isDaemonMode           bool
	clusterInstalledCRDs   []string
	isolateUserInNamespace bool
	tokenRetriever         *tokenRetriever
}

func newArenaConfiger(args types.ArenaClientArgs) (*ArenaConfiger, error) {
	tr := &tokenRetriever{}
	arenaConfigs, err := loadArenaConifg()
	if err != nil {
		return nil, err
	}
	clientConfig, restConfig, clientSet, dynamicClient, err := initKubeClient(args.Kubeconfig)
	if err != nil {
		return nil, err
	}
	restConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		tr.rountTripper = rt
		return tr
	})
	apiExtensionClientSet, err := extclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	/*
		crdNames, err := getClusterInstalledCRDs(apiExtensionClientSet)
		if err != nil {
			return nil, err
		}
	*/

	
	namespace := updateNamespace(args.Namespace, arenaConfigs, clientConfig)
	log.Debugf("current namespace is %v", namespace)

	userName, err := getUserName(namespace, clientConfig, restConfig, clientSet, tr)
	if err != nil {
		return nil, err
	}

	group, account, err := getAccountFromCert(clientConfig, *userName)
	if err != nil {
		return nil, err
	}

	log.Debugf("succeed to get user name: %v from client-go", *userName)
	userId := util.Md5(*userName)

	log.Debugf("the user id is %v", userId)
	data := getGlobalConfigFromConfigmap(args.ArenaNamespace, clientSet)
	adminUsers := getAdminUserFromConfigmap(data)
	i, err := isolateUserInNamespace(namespace, clientSet)
	if err != nil {
		return nil, err
	}
	log.Debugf("enable isolate user in namespace %v: %v", namespace, i)

	cluser := GetClusterInfo(clientConfig)

	return &ArenaConfiger{
		restConfig:             restConfig,
		clientConfig:           clientConfig,
		clientset:              clientSet,
		dynamicClient:          dynamicClient,
		apiExtensionClientset:  apiExtensionClientSet,
		namespace:              args.Namespace,
		arenaNamespace:         args.ArenaNamespace,
		configs:                arenaConfigs,
		isDaemonMode:           args.IsDaemonMode,
		clusterInstalledCRDs:   []string{},
		user:                   User{name: *userName, id: userId, group: group, account: account},
		cluster:                cluser,
		adminUsers:             adminUsers,
		isolateUserInNamespace: i,
		tokenRetriever:         tr,
	}, nil

}

func (a *ArenaConfiger) ToRESTConfig() (*rest.Config, error) {
	return a.restConfig, nil
}

func (a *ArenaConfiger) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(a.restConfig)
	if err != nil {
		return nil, err
	}

	cachedDiscoveryClient := memory.NewMemCacheClient(discoveryClient)
	return cachedDiscoveryClient, nil
}

func (a *ArenaConfiger) ToRESTMapper() (meta.RESTMapper, error) {

	cachedDiscoveryClient, err := a.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, cachedDiscoveryClient)
	return expander, nil
}

func (a *ArenaConfiger) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return a.clientConfig
}

// GetClientConfig returns the kubernetes ClientConfig
func (a *ArenaConfiger) GetClientConfig() clientcmd.ClientConfig {
	return a.clientConfig
}

// GetRestConfig returns the kubernetes RestConfig
func (a *ArenaConfiger) GetRestConfig() *rest.Config {
	return a.restConfig
}

// GetClientSet returns the kubernetes ClientSet
func (a *ArenaConfiger) GetClientSet() *kubernetes.Clientset {
	return a.clientset
}

// GetDynamicClient returns the kubernetes dynamic.Interface
func (a *ArenaConfiger) GetDynamicClient() dynamic.Interface {
	return a.dynamicClient
}

func (a *ArenaConfiger) GetAPIExtensionClientSet() *extclientset.Clientset {
	return a.apiExtensionClientset
}

// GetArenaNamespace returns the kubernetes namespace which some operators exists in
func (a *ArenaConfiger) GetArenaNamespace() string {
	return a.arenaNamespace
}

// GetNamespace returns the namespace of user assigns
func (a *ArenaConfiger) GetNamespace() string {
	return a.namespace
}

// GetConfigsFromConfigFile returns the configs read from config file
func (a *ArenaConfiger) GetConfigsFromConfigFile() map[string]string {
	return a.configs
}

func (a *ArenaConfiger) IsDaemonMode() bool {
	return a.isDaemonMode
}

func (a *ArenaConfiger) GetClusterInstalledCRDs() []string {
	return a.clusterInstalledCRDs
}

func (a *ArenaConfiger) GetUser() User {
	return a.user
}

func (a *ArenaConfiger) GetCluster() Cluster {
	return a.cluster
}

func (a *ArenaConfiger) IsIsolateUserInNamespace() bool {
	return a.isolateUserInNamespace
}

func (a *ArenaConfiger) GetAdminUsers() []User {
	return a.adminUsers
}

func (a *ArenaConfiger) IsAdminUser() bool {
	for _, admin := range a.adminUsers {
		if a.user.GetId() == admin.GetId() {
			return true
		}
	}
	return false
}

func getGlobalConfigFromConfigmap(namespace string, client *kubernetes.Clientset) map[string]string {
	data := map[string]string{}
	configmap, err := client.CoreV1().ConfigMaps(namespace).Get(context.TODO(), GlobalConfigmapName, metav1.GetOptions{})
	if err != nil {
		log.Debugf("failed to get arena global configmap %v,reason: %v", GlobalConfigmapName, err)
		return data
	}
	return configmap.Data
}

func getAdminUserFromConfigmap(data map[string]string) []User {
	users := []User{}
	val, ok := data[AdminUserKeyInConfigmap]
	if !ok {
		return users
	}
	userNames := strings.Split(val, ",")
	for _, name := range userNames {
		name = strings.Trim(name, " ")
		if name == "" {
			continue
		}
		userId := util.Md5(name)
		u := User{name: name, id: userId}
		log.Debugf("found admin user: %v", u)
		users = append(users, u)
	}
	return users
}

// loadArenaConifg returns configs in map
func loadArenaConifg() (map[string]string, error) {
	arenaConfigs := map[string]string{}
	log.Debugf("start to init arena config")
	validateFile := func(file string) bool {
		if file == "" {
			return false
		}
		_, err := os.Stat(file)
		if err != nil {
			log.Debugf("failed to get state of file %v,reason: %v,skip to handle it", file, err)
			return false
		}
		return true
	}
	configFileName := os.Getenv(RecommendedConfigPathEnvVar)
	defaultConfigFile, err := homedir.Expand(DefaultArenaConfigPath)
	if err != nil {
		return arenaConfigs, err
	}
	// if config file path read from env is invalid,read it from default path
	if !validateFile(configFileName) {
		configFileName = defaultConfigFile
	}
	// if config file is invalid,return null
	if !validateFile(configFileName) {
		return arenaConfigs, nil
	}
	arenaConfigs = config.ReadConfigFile(configFileName)
	log.Debugf("arena configs: %v", arenaConfigs)
	return arenaConfigs, nil
}

func updateNamespace(namespace string, arenaConfigs map[string]string, clientConfig clientcmd.ClientConfig) string {
	if namespace != "" {
		return namespace
	}
	log.Debugf("we need to update the namespace")
	if n, ok := arenaConfigs["namespace"]; ok {
		log.Debugf("read namespace %v from arena configuration file", n)
		return n
	}
	n, _, err := clientConfig.Namespace()
	if err == nil {
		log.Debugf("read namespace %v from kubeconfig", n)
		return n
	}
	log.Debugf("failed to read namespace from kubeconfig,we set the default namespace with 'default'")
	return "default"
}

func isolateUserInNamespace(namespaceName string, clientSet *kubernetes.Clientset) (bool, error) {
	namespace, err := clientSet.CoreV1().Namespaces().Get(context.TODO(), namespaceName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return namespace.Labels[types.MultiTenantIsolationLabel] == "true", nil
}

func getClusterInstalledCRDs(client *extclientset.Clientset) ([]string, error) {
	selectorListOpts := metav1.ListOptions{}

	list, err := client.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), selectorListOpts)
	if err != nil {
		return nil, err
	}
	crds := []string{}
	for _, crd := range list.Items {
		crds = append(crds, crd.Name)
	}
	return crds, nil
}

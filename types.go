package main

//import "time"
///
type HlmSpec map[string]any
// 
// 
// 
// 			Since the issue of 
// 			automatic mapping of Helmv3tpl 
// 			could not be resolved, 
// 			the type definition 
// 			of HlmSpec was directly modified.
// 
// 
// 
// type HlmSpec struct {
// 			Env string `json:"env,omitempty"`
// 			StsSingle bool `json:"stsSingle,omitempty"`
// 			Application string `json:"application,omitempty"`
// 			Cloudns string `json:"cloudns,omitempty"`
// 			Helmv3tpl *map[string]map[string]Releasees `json:"helmv3tpl,omitempty"`
// 			Coredns string `json:"coredns,omitempty"`/**/
// 			Global Global `json:"global,omitempty"`
// }
// type Releasees struct {
// 			Rle *[]map[string]any `json:"_role,omitempty"`
// 			RoleBindingspls *[]map[string]any `json:"_roleBinding+,omitempty"`
// 			_ *[]map[string]any `json:"_,omitempty"`
// 			Cjpls *[]map[string]any `json:"cj+,omitempty"`
// 			ClusterRole *[]map[string]any `json:"clusterRole,omitempty"`
// 			ClusterroleBindingpls *[]map[string]any `json:"clusterroleBinding+,omitempty"`
// 	Cmpls *[]struct {
// 			Files string `json:"files,omitempty"`
// 	} `json:"cm+,omitempty"`
// 			Crd *[]map[string]any `json:"crd,omitempty"`
// 			Deploypls *[]map[string]any `json:"deploy+,omitempty"`
// 			Drpls *[]map[string]any `json:"dr+,omitempty"`
// 	Gwpls *[]struct {
// 			Hosts string `json:"hosts,omitempty"`
// 		Vps *[]struct {
// 			Ptc string `json:"ptc,omitempty"`
// 			Number int32 `json:"number,omitempty"`
// 			Name string `json:"name,omitempty"`
// 		} `json:"vps,omitempty"`
// 			CredentialName string `json:"credentialName,omitempty"`
// 	} `json:"gw+,omitempty"`
// 	Hpa *[]struct {
// 			Max int32 `json:"max,omitempty"`
// 	} `json:"hpa,omitempty"`
// 			Nspls *[]map[string]any `json:"ns+,omitempty"`
// 	Pvpls *[]struct {
// 			StgSize string `json:"stgSize,omitempty"`
// 			AccMode string `json:"accMode,omitempty"`
// 			VolHandle string `json:"volHandle,omitempty"`
// 			Servers string `json:"servers,omitempty"`
// 			Paths string `json:"paths,omitempty"`
// 			Drivers string `json:"drivers,omitempty"`
// 			MntOpts *[]string `json:"mntOption,omitempty"`
// 	} `json:"pv+,omitempty"`
// 	Pvc *[]struct {
// 			AccMode string `json:"accMode,omitempty"`
// 			VolumeName string `json:"volumeName,omitempty"`
// 			StgSize string `json:"stgSize,omitempty"`
// 	} `json:"pvc,omitempty"`
// 	Sapls *[]struct {
// 			Annotations map[string]string `json:"annotations,omitempty"`
// 	} `json:"sa+,omitempty"`
// 	Sepls *[]struct {
// 			Ip string `json:"ip,omitempty"`
// 		Vps *[]struct {
// 			Ptc string `json:"ptc,omitempty"`
// 			Number int32 `json:"number,omitempty"`
// 			Name string `json:"name,omitempty"`
// 		} `json:"vps,omitempty"`
// 			Is bool `json:"is,omitempty"`
// 	} `json:"se+,omitempty"`
// 	Secretpls *[]struct {
// 		Data struct {
// 			Crt string `json:"tls.crt,omitempty"`
// 			Cacerts string `json:"cacerts.pem,omitempty"`
// 			Key string `json:"tls.key,omitempty"`
// 		} `json:"data,omitempty"`
// 			Files string `json:"files,omitempty"`
// 			// * `json:"*,omitempty"`
// 	} `json:"secret+,omitempty"`
// 	StorageClasspls *[]struct {
// 		Annotations struct {
// 			DefaultsStg bool `json:"storageclass.kubernetes.io/is-default-class,omitempty"` //onlyOne*******
// 		} `json:"annotations,omitempty"`
// 	} `json:"storageClass+,omitempty"`
// 	Svc *[]struct {
// 			ClusterIP string `json:"clusterIP,omitempty"`
// 		Vps *[]struct {
// 			Ptc string `json:"ptc,omitempty"`
// 			S int32 `json:"s,omitempty"`
// 			Number int32 `json:"number,omitempty"`
// 			T int32 `json:"t,omitempty"`
// 			Name string `json:"name,omitempty"`
// 		} `json:"vps,omitempty"`
// 			ExternalIPs *[]map[string]any `json:"externalIPs,omitempty"`
// 	} `json:"svc,omitempty"`
// 	Vspls *[]struct {
// 		Vps *[]struct {
// 			Ptc string `json:"ptc,omitempty"`
// 			S int32 `json:"s,omitempty"`
// 			Number int32 `json:"number,omitempty"`
// 			T int32 `json:"t,omitempty"`
// 			Name string `json:"name,omitempty"`
// 		} `json:"vps,omitempty"`
// 	} `json:"vs+,omitempty"`
// }
// type Global struct {
// 			Reg string `json:"reg,omitempty"`
// 			Label *map[string]string `json:"label,omitempty"`
// 			Tag string `json:"tag,omitempty"`
// }

func init() {
	//
}
package main

import (
	coresv1 "k8s.io/api/core/v1"
)

type S = map[string]map[string]map[string]struct {
	Rle                 map[string]any `yaml:"_role,omitempty"`
	RleBindings         map[string]any `yaml:"_roleBinding+,omitempty"`
	_                   map[string]any `yaml:"_,omitempty"`
	Job                 map[string]any `yaml:"cj+,omitempty"`
	ClusterRole         map[string]any `yaml:"clusterRole,omitempty"`
	ClusterroleBindings map[string]any `yaml:"clusterroleBinding+,omitempty"`
	Cms                 []struct {
		Files string `yaml:"files,omitempty"`
	} `yaml:"cm+,omitempty"`
	Crd map[string]any `yaml:"crd,omitempty"`
	Dpy *[]struct {
		ApiVersion          string                        `yaml:"apiVersion,omitempty"`
		Completions         int32                         `yaml:"completions,omitempty"`
		HostAliases         []coresv1.HostAlias           `yaml:"hostAliases,omitempty"`
		InitContainers      []coresv1.Container           `yaml:"initContainers,omitempty"`
		Annotations         map[string]string             `yaml:"annotations,omitempty"`
		Kinds               string                        `yaml:"kinds,omitempty"`
		PodManagementPolicy string                        `yaml:"podManagementPolicy,omitempty"`
		VolumeMounts        []coresv1.VolumeMount         `yaml:"volumeMounts,omitempty"`
		Image               string                        `yaml:"image,omitempty"`
		Env                 []coresv1.EnvVar              `yaml:"env,omitempty"`
		Command             []string                      `yaml:"command,omitempty"`
		Args                []string                      `yaml:"args,omitempty"`
		Resources           *coresv1.ResourceRequirements `yaml:"resources,omitempty"`
		// * `yaml:"*,omitempty"`
		Replicas *int32 `yaml:"replicas,omitempty"`
		Vps      []struct {
			Ptc    string `yaml:"ptc,omitempty"`
			Number int32  `yaml:"number,omitempty"`
			Name   string `yaml:"name,omitempty"`
		} `yaml:"vps,omitempty"`
		Path                 string                          `yaml:"path,omitempty"`
		Containers           coresv1.Container               `yaml:"containers,omitempty"`
		SecurityContext      *coresv1.PodSecurityContext     `yaml:"securityContext,omitempty"`
		VolumeClaimTemplates []coresv1.PersistentVolumeClaim `yaml:"volumeClaimTemplates,omitempty"`
		Volumes              []coresv1.Volume                `yaml:"volumes,omitempty"`
		StartupProbe         *coresv1.Probe                  `yaml:"startupProbe,omitempty"`
		Sid                  string                          `yaml:"sid,omitempty"`
		SecurityOpt          *coresv1.SecurityContext        `yaml:"security-opt,omitempty"`
		ServiceAccountName   string                          `yaml:"serviceAccountName,omitempty"`
		RestartPolicy        coresv1.RestartPolicy           `yaml:"restartPolicy,omitempty"`
		WorkingDir           string                          `yaml:"workingDir,omitempty"`
		EphemeralContainers  []coresv1.EphemeralContainer    `yaml:"ephemeralContainers,omitempty"`
		Affinity             *coresv1.Affinity               `yaml:"affinity,omitempty"`
		NodeSelector         map[string]string               `yaml:"nodeSelector,omitempty"`
		Tolerations          []coresv1.Toleration            `yaml:"tolerations,omitempty"`
	} `yaml:"deploy+,omitempty"`
	Drs map[string]any `yaml:"dr+,omitempty"`
	Gws []struct {
		Hosts string `yaml:"hosts,omitempty"`
		Vps   []struct {
			Ptc    string `yaml:"ptc,omitempty"`
			Number int32  `yaml:"number,omitempty"`
			Name   string `yaml:"name,omitempty"`
		} `yaml:"vps,omitempty"`
		CredentialName string `yaml:"credentialName,omitempty"`
	} `yaml:"gw+,omitempty"`
	Hpa []struct {
		Max int32 `yaml:"max,omitempty"`
	} `yaml:"hpa,omitempty"`
	Nss map[string]any `yaml:"ns+,omitempty"`
	Pvs []struct {
		StgSize   string           `yaml:"stgSize,omitempty"`
		AccMode   string           `yaml:"accMode,omitempty"`
		VolHandle string           `yaml:"volHandle,omitempty"`
		Servers   string           `yaml:"servers,omitempty"`
		Paths     string           `yaml:"paths,omitempty"`
		Drivers   string           `yaml:"drivers,omitempty"`
		MntOption []map[string]any `yaml:"mntOption,omitempty"`
	} `yaml:"pv+,omitempty"`
	Pvc []struct {
		AccMode    string `yaml:"accMode,omitempty"`
		VolumeName string `yaml:"volumeName,omitempty"`
		StgSize    string `yaml:"stgSize,omitempty"`
	} `yaml:"pvc,omitempty"`
	Sas []struct {
		Annotations string `yaml:"annotations,omitempty"`
	} `yaml:"sa+,omitempty"`
	Ses []struct {
		Ptc    string `yaml:"ptc,omitempty"`
		Number int32  `yaml:"number,omitempty"`
		Name   string `yaml:"name,omitempty"`
	} `yaml:"se+,omitempty"`
	Secrets []struct {
		data struct {
			Crt string `yaml:"tls.crt,omitempty"`
			Cas string `yaml:"cacerts.pem,omitempty"`
			Key string `yaml:"tls.key,omitempty"`
		} `yaml:"ptc,omitempty"`
		files string `yaml:"files,omitempty"`
		// * `yaml:"*,omitempty"`
	} `yaml:"secret+,omitempty"`
	StorageClasss []struct {
		annotations struct {
			DefaultsStg bool `yaml:"storageclass.kubernetes.io/is-default-class,omitempty"`
		} `yaml:"annotations,omitempty"`
	} `yaml:"storageClass+,omitempty"`
	Svc []struct {
		Ptc    string `yaml:"ptc,omitempty"`
		S      string `yaml:"s,omitempty"`
		Number int32  `yaml:"number,omitempty"`
		T      string `yaml:"t,omitempty"`
		Name   string `yaml:"name,omitempty"`
	} `yaml:"svc,omitempty"`
	Vss []struct {
		Ptc    string `yaml:"ptc,omitempty"`
		S      string `yaml:"s,omitempty"`
		Number int32  `yaml:"number,omitempty"`
		T      string `yaml:"t,omitempty"`
		Name   string `yaml:"name,omitempty"`
	} `yaml:"vs+,omitempty"`
}

type HlmSpec struct {
	Env       string `yaml:"env,omitempty"`
	StsSingle bool   `yaml:"stsSingle,omitempty"`
	App       string `yaml:"app,omitempty"`
	Cloudns   string `yaml:"cloudns,omitempty"`
	S
	Coredns string `yaml:"coredns,omitempty"`
	Global  struct {
		Reg   string            `yaml:"reg,omitempty"`
		Label map[string]string `yaml:"label,omitempty"`
		Tag   string            `yaml:"tag,omitempty"`
	} `yaml:"global,omitempty"`
}

func init() {
	//
}

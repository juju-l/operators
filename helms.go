package main

import (
			"os"
			"errors"
			"helm.sh/helm/v4/pkg/action"
			hcl "helm.sh/helm/v4/pkg/cli" //
			"helm.sh/helm/v4/pkg/cli/values"
			"helm.sh/helm/v4/pkg/chart/common"
			"helm.sh/helm/v4/pkg/chart/common/util"
			"helm.sh/helm/v4/pkg/chart/loader"
			"helm.sh/helm/v4/pkg/getter"
			"helm.sh/helm/v4/pkg/kube"
			"helm.sh/helm/v4/pkg/engine"
			//"helm.sh/helm/v4/pkg/release"
			"encoding/json"
			/*"*/
			"time"
)

var (
			// chartPth string
			actionCfg *action.Configuration
			// namespace string
			// releaseNme string
			// valueFle string
)

func helms(chartPth string, namespace,releaseNme,operators string, hlm *Hlm) (/*,*/map[string]any,error){
			if hlm.Spec == nil {
					return nil,/*,*/errors.New("empty spec")
			} else if true {
					b,_ := json.Marshal(hlm.Spec)
					valueFle := namespace+"-"+releaseNme+"-"+"operators"
					os.WriteFile(valueFle, b, 0600)
	var rst = make(map[string]any)
	settings := hcl.New()
	settings.SetNamespace(namespace)
	if err:=actionCfg.Init(
			settings.RESTClientGetter(), namespace, "secrets"/*os.Getenv("")*/,
		); err!=nil{
			return rst,/*,*/err
	} else if true {
			valOpts := &values.Options{
					ValueFiles: []string{valueFle},
				}
			if val,err:=valOpts.MergeValues(getter.Providers{}); err!=nil{
			return rst,/*,*/err
			} else if true {
				if charter,err:=loader.Load(chartPth); err!=nil{
					return rst,/*,*/err
				} else if true {
					if mgv,err:=util.ToRenderValues(charter, val, common.ReleaseOptions{Name: releaseNme,Namespace: namespace,Revision: 0,IsInstall: false,IsUpgrade: false}, nil, /*,*/); err!=nil{
						return rst,/*,*/err
					} else if true {
							if _/*manifests*/,err:=(&engine.Engine{}).Render(charter, mgv, /*,*/); err!=nil{
								return rst,/*,*/err
							} else if true {
		 switch operators {
			case "sta" :
				statuss := action.NewStatus(
					actionCfg,
					)
				/*statuss.WaitStrategy = kube.StatusWatcherStrategy*/
				rls,err:=statuss.Run(releaseNme)
				if err!=nil {
					return nil,/*,*/err
				}
				bytes,_ := json.Marshal(rls)
				err = json.Unmarshal(bytes, &rst)
				return rst,/*,*/err
			case "install" :
				install := action.NewInstall(
					actionCfg,
					)
				install.Namespace = namespace
				install.ReleaseName = releaseNme
				install.ServerSideApply = false
				install.ForceReplace = true
				install.ForceConflicts = true
				install.WaitStrategy = kube.StatusWatcherStrategy
				install.Timeout = 7*time.Minute
				rls,err:=install.Run(/*,*/ charter, val)
				if err!=nil {
					return nil,/*,*/err
				}
				bytes,_ := json.Marshal(rls)
				err = json.Unmarshal(bytes, &rst)
				return rst,/*,*/err
			case "rollbak" :
				rollbak := action.NewRollback(
					actionCfg,
					)
				rollbak.ServerSideApply = "false"
				rollbak.ForceReplace = true
				rollbak.ForceConflicts = true
				rollbak.WaitStrategy = kube.StatusWatcherStrategy
				rollbak.Timeout = 7*time.Minute
				/*rls,*/err:=rollbak.Run(releaseNme)
				if err!=nil {
					return nil,/*,*/err
				}
				//bytes,_ := json.Marshal(rls)
				//err = json.Unmarshal(bytes, &rst)
				return rst,/*,*/err
			case "upgrade" :
				upgrade := action.NewUpgrade(
					actionCfg,
					)
				upgrade.Namespace = namespace
				// upgrade.ReleaseName = releaseNme
				upgrade.ServerSideApply = "false"
				upgrade.ForceReplace = true
				upgrade.ForceConflicts = true
				upgrade.WaitStrategy = kube.StatusWatcherStrategy
				upgrade.Timeout = 7*time.Minute
				rls,err:=upgrade.Run(releaseNme, charter, val)
				if err!=nil {
					return nil,/*,*/err
				}
				bytes,_ := json.Marshal(rls)
				err = json.Unmarshal(bytes, &rst)
				return rst,/*,*/err
			case "rmv" :
				uninstall := action.NewUninstall(
					actionCfg,
					)
				uninstall.WaitStrategy = kube.StatusWatcherStrategy
				rls,err:=uninstall.Run(releaseNme)
				if err!=nil {
					return nil,/*,*/err
				}
				bytes,_ := json.Marshal(rls)
				err = json.Unmarshal(bytes, &rst)
				return rst,/*,*/err
			///for k,v:=range manifests{ rst[k] = /*""+"---\n"+*/v //sliceYml }
		 }
							} else {
								//
							}
					} else {
						//
					}
				} else {
					///helm.sh/zh/docs/sdk/gosdk; github.com/helm/helm-www/tree/main/sdkexamples; pkg.go.dev/helm.sh/helm/v4/pkg
				}
			} else {
			//
			}
			///
			//
			/////
	} else {
			return rst,/*,*/nil
	}
	//
	//
	//
					//
					///
					return rst,/*,*/nil
			} else {
					return nil,/*,*/nil
			}
}

func t() {
			// helms(chartPth,namespace,releaseNme,valueFle,"install") //
}

func init() {
			// chartPth = "tpl_vipex_cc-0.1.0.tgz"
			actionCfg = &action.Configuration{}
			// namespace = "default"
			// releaseNme = "helmv3tpl"
			// valueFle = "samples.yaml"
}

// var
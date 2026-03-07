
```
mkdir helms
cd ./helms
operator-sdk init --domain=vipex.cc --project-name= --plugins=helm
operator-sdk create api --group=tpl --version=v1alpha1 --kind=DpTools --plugins=helm
\cp -rfv ../tpl.vipex.cc helm-tpl.vipex.cc-charts
rm -rfv helm-charts
sed -E 's,helm-charts/dptools,./helm-tpl.vipex.cc-charts,g' -i watches.yaml
touch README.md
echo -n >z.md
```



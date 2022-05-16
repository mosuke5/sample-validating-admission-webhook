[![application ci](https://github.com/mosuke5/sample-validation-admission-webhook/actions/workflows/test.yaml/badge.svg)](https://github.com/mosuke5/sample-validation-admission-webhook/actions/workflows/test.yaml)

# Sample Validation Admission Webhook
This is a sample validation admission webhook.  
Specification is that if `.spec.SecurityContext.RunAsUser` is empty or root(value is `0`), refuse request, except namespace name matches `admin-*`.

このレポジトリはValidating Admission Webhookのサンプルです。  
仕様はとてもシンプルで、`.spec.SecurityContext.RunAsUser`が空かrootであるPodの作成を拒否します。ただし、Podの作成が `admin-*` の名前にマッチするNamespaceの場合は除外します。
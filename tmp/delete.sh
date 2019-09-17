#!/bin/bash
kubectl delete deploy,svc cw-maysunliberty2-6c1b1ce0-cb4c-11e9-be96 proja-reusable-build-container
kubectl delete job codewind-liberty-build-job

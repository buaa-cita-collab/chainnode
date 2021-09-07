#!/bin/bash
export CITA_CLOUD_RPC_ADDR=`minikube ip`:30004 CITA_CLOUD_EXECUTOR_ADDR=`minikube ip`:30005
for a in {1..100}
do
	cldi block-number
	sleep 0.2
done

# Yurt-device-controller Tutorial

This document introduces how to install yurt-device-controller and use yurt-device-controller to manage edge devices on edgex foundry. Suppose you have an OpenYurt cluster, and already have one edgex foundry instance in a NodePool.

## Environment

1. For details on setting up an OpenYurt cluster, please check out the [tutorial](https://github.com/openyurtio/openyurt#getting-started) ;

2. For details on deploying Yurt-app-Manager, please check out the [tutorial](https://github.com/openyurtio/yurt-app-manager/blob/master/docs/yurt-app-manager-tutorial.md) ;

3. For details on deploying the Edgex Foundry instance using yurt-edgex-manager, please check out the [tutorial](https://github.com/openyurtio/yurt-edgex-manager/blob/main/README.md) .

## Install yurt-device-controller

Suppose you have an OpenYurt cluster, a nodePool, and deployed an Edgex Foundry Instance on that node pool.

### Install the related CRDs

```bash
$ cd yurt-device-controller
$ kubectl apply -f config/setup/crd.yaml
```

### Deploy yurt-device-controller by using unitedDeployment

The following unitedDeployment example deploy a yurt-device-controller in hangzhou nodePool:

```bash
$ cat <<EOF | kubectl apply -f -
apiVersion: apps.openyurt.io/v1alpha1
kind: UnitedDeployment
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: ud-device
spec:
  selector:
    matchLabels:
      app: ud-device
  workloadTemplate:
    deploymentTemplate:
      metadata:
        labels:
          app: ud-device
      spec:
        template:
          metadata:
            labels:
              app: ud-device
              control-plane: controller-manager
          spec:
            containers:
            - args:
              - --health-probe-bind-address=:8081
              - --metrics-bind-address=127.0.0.1:8080
              - --leader-elect=false
              command:
              - /manager
              image: openyurt/yurt-device-controller:latest
              imagePullPolicy: IfNotPresent
              livenessProbe:
                httpGet:
                  path: /healthz
                  port: 8081
                initialDelaySeconds: 15
                periodSeconds: 20
              name: manager
              readinessProbe:
                httpGet:
                  path: /readyz
                  port: 8081
                initialDelaySeconds: 5
                periodSeconds: 10
              resources:
                limits:
                  cpu: 100m
                  memory: 30Mi
                requests:
                  cpu: 100m
                  memory: 20Mi
              securityContext:
                allowPrivilegeEscalation: false
            securityContext:
              runAsUser: 65532
            terminationGracePeriodSeconds: 10
  topology:
    pools:
    - name: hangzhou 
      nodeSelectorTerm:
        matchExpressions:
        - key: apps.openyurt.io/nodepool
          operator: In
          values:
          - hangzhou 
      replicas: 1
      tolerations:
      - effect: NoSchedule
        key: apps.openyurt.io/example
        operator: Exists
  revisionHistoryLimit: 5
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: default-cluster-admin
subjects:
- kind: ServiceAccount
  name: default
  namespace: default
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: ""
---
EOF
```

## How to use

This tutorial shows how to manipulate object instances in the Edgex Foundry by using yurt-device-controller.

### Create Device, DeviceService, DeviceProfile

1. Create a deviceService

   ```bash
   $ cat <<EOF | kubectl apply -f -
   apiVersion: device.openyurt.io/v1alpha1
   kind: DeviceService
   metadata:
     name: hangzhou-device-service-test
   spec:
     nodePool: hangzhou
     addressable:
       address: edgex-device-virtual-test
       method: POST
       name: device-service-test
       path: /api/v1/callback
       port: 49999
       protocol: HTTP
     adminState: UNLOCKED
     operatingState: ENABLED
   EOF
   ```



2. Create a deviceProfile

   ```bash
   $ cat <<EOF | kubectl apply -f -
   apiVersion: device.openyurt.io/v1alpha1
   kind: DeviceProfile
   metadata:
     name: hangzhou-sensor-test
   spec:
     nodePool: hangzhou
     description: Sensor cluster providing metrics for temperature and humidity
     manufacturer: Raspberry Pi Foundation
     model: Raspberry Pi 3b+
     labels:
     - rpi
     deviceResources:
     - name: temperature
       description: Sensor cluster temperature values
       properties:
         value:
           type: "Int64"
           readWrite: "RW"
           minimum: "-50"
           maximum: "100"
           size: "4"
           defaultValue: "9"
     - name: humidity
       description: Sensor cluster humidity values
       properties:
         value:
           type: "Int64"
           readWrite: "RW"
           minimum: "0"
           maximum: "100"
           size: "4"
           defaultValue: "9"
   EOF
   ```



3. Create a device

   Create a device using the deviceService and deviceProfile created above:

   ```sh
   $ cat <<EOF | kubectl apply -f -
   apiVersion: device.openyurt.io/v1alpha1
   kind: Device
   metadata:
     name: hangzhou-sample-device
   spec:
     nodePool: hangzhou
     adminState: UNLOCKED
     description: RESTful Device that sends in JSON data
     labels:
     - rest
     - json
     operatingState: ENABLED
     profile: hangzhou-sensor-test
     service: hangzhou-device-service-test
     protocols:
       other: {}
   EOF
   ```



### Delete Device, DeviceService, DeviceProfile

The deletion operation is very simple, you can delete device, deviceService and deviceProfile just like deleting ordinary K8S objects:

```bash
$ kubectl delete device hangzhou-sample-device
device.device.openyurt.io "hangzhou-sample-device" deleted

$ kubectl delete deviceservice hangzhou-device-service-test
deviceservice.device.openyurt.io "hangzhou-device-service-test" deleted

$ kubectl delete deviceprofile hangzhou-sensor-test
deviceprofile.device.openyurt.io "hangzhou-sensor-test" deleted
```

### Update the properties of device

The following operation uses `random-boolean-device` device, which is automatically created by the Edgex Foundry instance. This device will randomly generate bool values.

1. Check the status of device

   ```bash
   $ kubectl get device
   NAME                             AGE
   hangzhou-random-boolean-device   23h
   hangzhou-random-integer-device   23h
   
   $ kubectl describe device hangzhou-random-boolean-device
   Name:         hangzhou-random-boolean-device
   Namespace:    default
   Labels:       device-controller/edgex-object.name=random-boolean-device
   Annotations:  <none>
   API Version:  device.openyurt.io/v1alpha1
   Kind:         Device
   Metadata:
     Creation Timestamp:  2021-09-22T06:03:04Z
     Finalizers:
       v1alpha1.device.finalizer
     # ......
     # ......
   Spec:
     Admin State:  UNLOCKED
     Description:  Example of Device Virtual
     Labels:
       device-virtual-example
     Managed:          false
     Node Pool:        hangzhou
     Operating State:  ENABLED
     Profile:          Random-Boolean-Device
     Protocols:
       Other:
         Address:  device-virtual-bool-01
         Port:     300
     Service:      device-virtual
   Status:
     Admin State:  UNLOCKED
     Conditions:
       Last Transition Time:  2021-09-23T05:36:29Z
       Reason:                this device is not managed by openyurt
       Severity:              Info
       Status:                False
       Type:                  Ready
       Last Transition Time:  2021-09-23T05:36:29Z
       Reason:                this device is not managed by openyurt
       Severity:              Info
       Status:                False
       Type:                  DeviceManaging
       Last Transition Time:  2021-09-22T06:03:05Z
       Status:                True
       Type:                  DeviceSynced
     Device Properties:
       Bool:
         Actual Value:  true
         Get URL:       http://edgex-core-command:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/7369b8b1-6772-420b-93d2-b3cedef9c50b
         Name:          Bool
       Bool Array:
         Actual Value:  [false,false,false,false,false]
         Get URL:       http://edgex-core-command:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/e1756728-8f8d-444d-a9a0-f3168a294923
         Name:          BoolArray
     Edge Id:           07b0d343-cc07-43ff-afb1-6a2792d48b7f
     Operating State:   ENABLED
     Synced:            true
   Events:              <none>
   ```



2. Set the `managed` field of device to `true`

   The `Device.Spec.Managed` field determines whether the cloud can set the property value of the edge device. The cloud can successfully set device properties only if `managed=true`:

   ```bash
   $ kubectl patch devices.device.openyurt.io hangzhou-random-boolean-device -p '{"spec":{"managed":true}}'  --type=merge
   ```

   The device status after field `managed` is set to `true`:

   ```yaml
   Status:
     Admin State:  UNLOCKED
     Conditions:
       Last Transition Time:  2021-09-23T06:09:22Z
       Status:                True
       Type:                  Ready
       Last Transition Time:  2021-09-23T06:09:22Z
       Status:                True
       Type:                  DeviceManaging
       Last Transition Time:  2021-09-22T06:03:05Z
       Status:                True
       Type:                  DeviceSynced
     Device Properties:
       Bool:
         Actual Value:  true
         Get URL:       http://edgex-core-command:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/7369b8b1-6772-420b-93d2-b3cedef9c50b
         Name:          Bool
       Bool Array:
         Actual Value:  [false,true,false,true,true]
         Get URL:       http://edgex-core-command:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/e1756728-8f8d-444d-a9a0-f3168a294923
         Name:          BoolArray
     Edge Id:           07b0d343-cc07-43ff-afb1-6a2792d48b7f
     Operating State:   ENABLED
     Synced:            true
   Events:              <none>
   ```

3. Change the `adminState` of device

   > The administrative state (aka admin state) provides control of the device service by man or other systems. It can be set to LOCKED or UNLOCKED. When a device service is set to locked, it is not suppose to respond to any command requests nor send data from the devices.

   ```bash
   $ kubectl patch devices.device.openyurt.io hangzhou-random-boolean-device -p '{"spec":{"adminState":"LOCKED"}}'  --type=merge
   ```

   The device status after field `adminState` is set to `LOCKED`:

   ```yaml
   Status:
     Admin State:  LOCKED
     Conditions:
       Last Transition Time:  2021-09-23T06:09:22Z
       Status:                True
       Type:                  Ready
       Last Transition Time:  2021-09-23T06:09:22Z
       Status:                True
       Type:                  DeviceManaging
       Last Transition Time:  2021-09-22T06:03:05Z
       Status:                True
       Type:                  DeviceSynced
     Device Properties:
       Bool:
         Actual Value:
         Get URL:       http://edgex-core-command:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/c09f856a-36cf-4d95-a352-e42cf8d451dc
         Name:          Bool
       Bool Array:
         Actual Value:
         Get URL:       http://edgex-core-command:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/49618976-19ce-4084-84ea-90428d57c547
         Name:          BoolArray
     Edge Id:           07b0d343-cc07-43ff-afb1-6a2792d48b7f
     Operating State:   ENABLED
     Synced:            true
   ```



4. Change the `operatingState` of device

   > The operating state (aka op state) provides an indication on the part of EdgeX about the internal operating status of the device service. The operating state is not set externally (as by another system or man), it is a signal from within EdgeX (and potentially the device service itself) about the condition of the service. The operating state of the device service may be either enabled or disabled. When the operating state of the device service is disabled, it is either experiencing some difficulty or going through some process (for example an upgrade) which does not allow it to function in its normal capacity.

   ```bash
   $ kubectl patch devices.device.openyurt.io hangzhou-random-boolean-device -p '{"spec":{"operatingState":"DISABLED"}}'  --type=merge
   ```

   The device status after field `operatingState` is set to `DISABLED`:

   ```yaml
   Status:
     Admin State:  UNLOCKED
     Conditions:
       Last Transition Time:  2021-09-23T06:09:22Z
       Status:                True
       Type:                  Ready
       Last Transition Time:  2021-09-23T06:09:22Z
       Status:                True
       Type:                  DeviceManaging
       Last Transition Time:  2021-09-22T06:03:05Z
       Status:                True
       Type:                  DeviceSynced
     Edge Id:                 07b0d343-cc07-43ff-afb1-6a2792d48b7f
     Operating State:         DISABLED
     Synced:                  true
   ```



5. Set the deviceProperties

   ```bash
   # Ensure adminState = UNLOCKED
   $ kubectl patch devices.device.openyurt.io hangzhou-random-boolean-device -p '{"spec":{"adminState":"UNLOCKED"}}'  --type=merge
   
   # Ensure operatingState = ENABLED
   $ kubectl patch devices.device.openyurt.io hangzhou-random-boolean-device -p '{"spec":{"operatingState":"ENABLED"}}'  --type=merge
   ```

   Because the `putURL` of random-boolean-device's bool property may change after `adminState` or `operingState` is changed. Therefore, you need to query the deviceProperty status again to obtain the latest `putURL`:

   ```bash
   $ kubectl describe device hangzhou-random-boolean-device
   ```

   The status is :

   ```yaml
   # ....
   Status:
     Admin State:  UNLOCKED
     Conditions:
       Last Transition Time:  2021-09-23T06:24:02Z
       Status:                True
       Type:                  Ready
       Last Transition Time:  2021-09-23T06:24:02Z
       Status:                True
       Type:                  DeviceManaging
       Last Transition Time:  2021-09-22T06:03:05Z
       Status:                True
       Type:                  DeviceSynced
     Device Properties:
       Bool:
         Actual Value:  true
         Get URL:       http://edgex-core-command:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/9a61a8d5-7c15-4d1b-b552-15b7879d9fc8
         Name:          Bool
       Bool Array:
         Actual Value:  [true,false,true,true,true]
         Get URL:       http://edgex-core-command:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/5407cb4f-0f99-47dc-8cce-3fd39cedaddf
         Name:          BoolArray
     Edge Id:           07b0d343-cc07-43ff-afb1-6a2792d48b7f
     Operating State:   ENABLED
     Synced:            true
   ```

   Set the `Bool` property of the device to ` false`:

   ```bash
   $ kubectl patch devices.device.openyurt.io hangzhou-random-boolean-device  --type=merge -p '{
   "spec":{
     "deviceProperties":{
       "Bool":{
         "desiredValue":"false", 
         "name":"Bool", 
         "putURL":"http://edgex-core-command:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/9a61a8d5-7c15-4d1b-b552-15b7879d9fc8"
       }
     }
   }
   }'
   ```

   Check the status of deviceProperties in the Edgex Foundry:

   ```bash
   $ kubectl get service | grep edgex-core-command
   edgex-core-command                     NodePort    10.96.39.34     <none>        48082:30082/TCP                  39h
   
   $ curl http://10.96.39.34:48082/api/v1/device/07b0d343-cc07-43ff-afb1-6a2792d48b7f/command/9a61a8d5-7c15-4d1b-b552-15b7879d9fc8
   {"device":"random-boolean-device","origin":1632378327952106491,"readings":[{"origin":1632378327951971484,"device":"random-boolean-device","name":"Bool","value":"false","valueType":"Bool"}],"EncodedEvent":null}
   ```


**Caution**: This tutorial is dedicated for OpenYurt and EdgeX Foundry 2.x, for OpenYurt with EdgeX Foundry 1.x,
please refer to [legacy v1 tutorial](https://github.com/openyurtio/yurt-device-controller/blob/main/docs/yurt-device-controller-tutorial-v1.md)

# Yurt-device-controller Tutorial

This document introduces how to install yurt-device-controller and use yurt-device-controller to manage edge devices on EdgeX Foundry V2. We suppose you have an OpenYurt cluster, and already have one EdgeX Foundry V2 instance deployed in a NodePool.

## Environment

1. For details on setting up an OpenYurt cluster, please check out the [tutorial](https://github.com/openyurtio/openyurt#getting-started) ;

2. For details on deploying Yurt-app-Manager, please check out the [tutorial](https://github.com/openyurtio/yurt-app-manager/blob/master/docs/yurt-app-manager-tutorial.md) ;

3. For details on deploying the EdgeX Foundry instance using yurt-edgex-manager, please check out the [tutorial](https://github.com/openyurtio/yurt-edgex-manager/blob/main/README.md) .

## Install yurt-device-controller

Suppose you have an OpenYurt cluster, a NodePool, and deployed an EdgeX Foundry Instance on that node pool.

### Register OpenYurt device management related CRDs

The following bash command will register Device, DeviceProfile and DeviceService CRDs into the cluster:

```shell
$ cd yurt-device-controller
$ kubectl apply -f config/setup/crd.yaml
```

### Deploy yurt-device-controller using UnitedDeployment

Suppose you have already created a NodePool named hangzhou. Then the following UnitedDeployment example deploys a
yurt-device-controller in this NodePool. It should be pointed out that we use `cluster-admin` ClusterRole just for demo purpose.

```yaml
apiVersion: apps.openyurt.io/v1alpha1
kind: UnitedDeployment
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: ud-device
  namespace: default
spec:
  selector:
    matchLabels:
      app: ud-device
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
      - operator: Exists
  workloadTemplate:
    deploymentTemplate:
      metadata:
        creationTimestamp: null
        labels:
          app: ud-device
      spec:
        selector:
          matchLabels:
            app: ud-device
        strategy: {}
        template:
          metadata:
            creationTimestamp: null
            labels:
              app: ud-device
              control-plane: controller-manager
          spec:
            containers:
            - args:
              - --health-probe-bind-address=:8081
              - --metrics-bind-address=127.0.0.1:8080
              - --leader-elect=false
              - --v=5
              command:
              - /yurt-device-controller
              image: openyurt/yurt-device-controller:v0.2.0
              imagePullPolicy: IfNotPresent
              livenessProbe:
                failureThreshold: 3
                httpGet:
                  path: /healthz
                  port: 8081
                  scheme: HTTP
                initialDelaySeconds: 15
                periodSeconds: 20
                successThreshold: 1
                timeoutSeconds: 1
              name: manager
              readinessProbe:
                failureThreshold: 3
                httpGet:
                  path: /readyz
                  port: 8081
                  scheme: HTTP
                initialDelaySeconds: 5
                periodSeconds: 10
                successThreshold: 1
                timeoutSeconds: 1
              resources:
                limits:
                  cpu: 100m
                  memory: 512Mi
                requests:
                  cpu: 100m
                  memory: 512Mi
              securityContext:
                allowPrivilegeEscalation: false
            dnsPolicy: ClusterFirst
            restartPolicy: Always
            securityContext:
              runAsUser: 65532
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ud-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: default
  namespace: default
```

## How to use

The following section will show you how to manage devices in EdgeX Foundry according to OpenYurt CRDs.

### Simulate leaf devices in EdgeX

Our trip starts with simulating various kinds of leaf devices. To make things easy, we just deploy a virtual device
driver [device-virtual-go](https://github.com/edgexfoundry/device-virtual-go). It simulates different kinds of devices
to generate device data, and users can send commands to get responses from or conduct control instructions to the devices.

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    org.edgexfoundry.service: edgex-device-virtual
  name: edgex-device-virtual
spec:
  replicas: 1
  selector:
    matchLabels:
      org.edgexfoundry.service: edgex-device-virtual
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        org.edgexfoundry.service: edgex-device-virtual
    spec:
      hostname: edgex-device-virtual
      nodeSelector:
        apps.openyurt.io/nodepool: hangzhou
      containers:
      - name: edgex-device-virtual
        image: openyurt/device-virtual:2.1.0
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 59900
          name: "tcp-59900"
          protocol: TCP
        env:
        - name: MESSAGEQUEUE_HOST
          value: edgex-redis
        - name: SERVICE_HOST
          value: edgex-device-virtual
        envFrom:
        - configMapRef:
            name: common-variables
        startupProbe:
          tcpSocket:
            port: 59900
          periodSeconds: 1
          failureThreshold: 120
        livenessProbe:
          tcpSocket:
            port: 59900
      restartPolicy: Always
---
apiVersion: v1
kind: Service
metadata:
  labels:
    org.edgexfoundry.service: edgex-device-virtual
  name: edgex-device-virtual
spec:
  ports:
  - name: "tcp-59900"
    port: 59900
    protocol: TCP
    targetPort: 59900
  selector:
    org.edgexfoundry.service: edgex-device-virtual
  type: NodePort
```

The device-virtual-go driver will automatically create and register 5 virtual devices of different kinds upon start,
yurt-device-controller will then sync them to OpenYurt. You can use kubectl to check it:

```shell
$ kubectl get device
NAME                                     NODEPOOL   SYNCED   AGE
hangzhou-random-binary-device            hangzhou   true     19h
hangzhou-random-boolean-device           hangzhou   true     19h
hangzhou-random-float-device             hangzhou   true     19h
hangzhou-random-integer-device           hangzhou   true     19h
hangzhou-random-unsignedinteger-device   hangzhou   true     19h
```

### Create Device, DeviceService, DeviceProfile

1. Create a DeviceService

```yaml
apiVersion: device.openyurt.io/v1alpha1
kind: DeviceService
metadata:
  name: openyurt-created-deviceservice-virtual
spec:
  adminState: UNLOCKED
  baseAddress: http://edgex-device-virtual:59900
  nodePool: hangzhou
```

**Note**: The `baseAddress` field is important here, it means the address of the backend driver to communicate with devices.
So here, we set the value to the service address of `device-virtual-go` we deployed earlier.

2. Create a DeviceProfile

```yaml
apiVersion: device.openyurt.io/v1alpha1
kind: DeviceProfile
metadata:
  name: openyurt-created-random-boolean-deviceprofile
spec:
  description: Example of Device-Virtual Created By OpenYurt
  deviceCommands:
  - isHidden: false
    name: WriteBoolValue
    readWrite: W
    resourceOperations:
    - defaultValue: ""
      deviceResource: Bool
    - defaultValue: "false"
      deviceResource: EnableRandomization_Bool
  - isHidden: false
    name: WriteBoolArrayValue
    readWrite: W
    resourceOperations:
    - defaultValue: ""
      deviceResource: BoolArray
    - defaultValue: "false"
      deviceResource: EnableRandomization_BoolArray
  deviceResources:
  - description: used to decide whether to re-generate a random value
    isHidden: true
    name: EnableRandomization_Bool
    properties:
      defaultValue: "true"
      readWrite: W
      valueType: Bool
  - description: Generate random boolean value
    isHidden: false
    name: Bool
    properties:
      defaultValue: "true"
      readWrite: RW
      valueType: Bool
  - description: used to decide whether to re-generate a random value
    isHidden: true
    name: EnableRandomization_BoolArray
    properties:
      defaultValue: "true"
      readWrite: W
      valueType: Bool
  - description: Generate random boolean array value
    isHidden: false
    name: BoolArray
    properties:
      defaultValue: '[true]'
      readWrite: RW
      valueType: BoolArray
  labels:
  - openyurt-created-device-virtual-example
  manufacturer: OpenYurt Community
  model: OpenYurt-Device-Virtual-01
  nodePool: hangzhou
```

This DeviceProfile is just a copy of `random-boolean` DeviceService created by `device-virtual-go` for demo purpose.

3. Create a Device

Create a virtual device using the DeviceService and DeviceProfile created above:

```yaml
apiVersion: device.openyurt.io/v1alpha1
kind: Device
metadata:
  name: openyurt-created-random-boolean-device
spec:
  adminState: UNLOCKED
  description: Example of Device Virtual
  labels:
  - openyurt-created-device-virtual-example
  managed: true
  nodePool: hangzhou
  notify: true
  operatingState: UP
  profileName: openyurt-created-random-boolean-deviceprofile
  protocols:
    other:
      Address: openyurt-created-device-virtual-bool-01
      Port: "300"
  serviceName: openyurt-created-deviceservice-virtual
```

Then, we can see the resource objects in OpenYurt through kubectl as below:

```shell
$ kubectl get deviceservice  openyurt-created-deviceservice-virtual
NAME                                     NODEPOOL   SYNCED   AGE
openyurt-created-deviceservice-virtual   hangzhou   true     14h

$ kubectl get deviceprofile openyurt-created-random-boolean-deviceprofile
NAME                                            NODEPOOL   SYNCED   AGE
openyurt-created-random-boolean-deviceprofile   hangzhou   true     15h

$ kubectl get device openyurt-created-random-boolean-device
NAME                                     NODEPOOL   SYNCED   AGE
openyurt-created-random-boolean-device   hangzhou   true     14h
```

### Retrieve device generated data

We have already set up the environment and simulated a virtual bool device. In OpenYurt, we can easily get the latest
data generated by devices just by checking the `status` sub-resource of Device resource object like this:

```shell
$ kubectl get device openyurt-created-random-boolean-device -o yaml
apiVersion: device.openyurt.io/v1alpha1
kind: Device
metadata:
  creationTimestamp: "2022-05-30T12:19:07Z"
  finalizers:
  - v1alpha1.device.finalizer
  generation: 2
  name: openyurt-created-random-boolean-device
  namespace: default
  resourceVersion: "1454496"
  uid: 3035fca2-0183-4f85-ab7a-23d8e2841166
spec:
  adminState: UNLOCKED
  description: Example of Device Virtual
  deviceProperties:
    Bool:
      desiredValue: "true"
      name: Bool
  labels:
  - openyurt-created-device-virtual-example
  managed: false
  nodePool: hangzhou
  notify: true
  operatingState: UP
  profileName: openyurt-created-random-boolean-deviceprofile
  protocols:
    other:
      Address: openyurt-created-device-virtual-bool-01
      Port: "300"
  serviceName: openyurt-created-deviceservice-virtual
status:
  adminState: UNLOCKED
  conditions:
  - lastTransitionTime: "2022-05-30T18:30:34Z"
    status: "True"
    type: Ready
  - lastTransitionTime: "2022-05-30T18:30:34Z"
    status: "True"
    type: DeviceManaging
  - lastTransitionTime: "2022-05-30T12:19:07Z"
    status: "True"
    type: DeviceSynced
  deviceProperties:
    Bool:
      actualValue: "false"
      getURL: http://edgex-core-command:59882/api/v2/device/name/openyurt-created-random-boolean-device/Bool
      name: Bool
    BoolArray:
      actualValue: '[false, true, true, true, false]'
      getURL: http://edgex-core-command:59882/api/v2/device/name/openyurt-created-random-boolean-device/BoolArray
      name: BoolArray
  edgeId: 01c87af0-1a0f-4227-b68d-4c03de19f25b
  operatingState: UP
  synced: true
```

The `deviceProperties` shows all the properties of this device. For example, the `Bool` property has the latest value `false`
and the value is retrieved from the EdgeX rest api `http://edgex-core-command:59882/api/v2/device/name/openyurt-created-random-boolean-device/Bool`.

### Update the properties of device

If you want to control a device by updating its writable property, you should first set `Device.Spec.Managed` field to
`true` to indicate yurt-device-controller, otherwise all the update operations will be ignored.

1. Set the `managed` field of device to `true`

```shell
$ kubectl patch device openyurt-created-random-boolean-device -p '{"spec":{"managed":true}}'  --type=merge
```

3. Change the `adminState` of device

   > The administrative state (aka admin state) provides control of the device service by man or other systems. It can
   > be set to `LOCKED` or `UNLOCKED`. When a device service is set to locked, it is not supposed to respond to any
   > command requests nor send data from the devices.

```shell
$ kubectl patch device openyurt-created-random-boolean-device -p '{"spec":{"adminState":"UNLOCKED"}}'  --type=merge
```

4. Set the DeviceProperties to control/update device

```shell
kubectl patch device openyurt-created-random-boolean-device --type=merge -p '{"spec":{"managed":true,"deviceProperties":{"Bool": {"name":"Bool", "desiredValue":"true"}}}}'
```

In the command, we set the `Bool` DeviceProperty value to `true`, yurt-device-controller will trigger a EdgeX command
and change the property of the device. We can check this by watch the status of device for multiple times, you will find
the value is always `true` unless you change this property to `false` again.

```shell
watch "kubectl get device openyurt-created-random-boolean-device -o json | jq '.status.deviceProperties.Bool.actualValue'"

# output
Every 2.0s: kubectl get device openyurt-created-random-boolean-device -o json | jq '.status.deviceProperties.Bool.actualValue'                                                                                                                                             wawlian-m1.local: Tue May 31 11:47:36 2022

"true"
```

### Delete Device, DeviceService, DeviceProfile

The deletion operation is really simple, you can delete device, deviceService and deviceProfile just like deleting ordinary K8S resource objects:

```shell
$ kubectl delete device openyurt-created-random-boolean-device
device.device.openyurt.io "openyurt-created-random-boolean-device" deleted

$ kubectl delete deviceservice openyurt-created-deviceservice-virtual
deviceservice.device.openyurt.io "openyurt-created-deviceservice-virtual" deleted

$ kubectl delete deviceprofile openyurt-created-random-boolean-deviceprofile
deviceprofile.device.openyurt.io "openyurt-created-random-boolean-deviceprofile" deleted
```

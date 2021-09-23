# Yurt-device-controller

This repository contains three CRD/controllers, Device, DeviceService and DeviceProfile:

- The `DeviceProfile` defines a type of devices using same kind of protocol, which includes some generic information like the manufacturer's name, the device description, and the device model. DeviceProfile also defines what kind of resources (e.g., temperature, humidity) this type of device provided and how to read/write these resources.

- The `DeviceService` defines the way of how to connect a device to the OpenYurt, like the URL of the device. The `DeviceService` can not exist alone. Every `DeviceService` must associate with a `DeviceProfile`.

- The `Device` is used to refer to a sensor, actuator, or IoT "thing", it gives the detailed definition of a specific device, like which `DeviceProfile` it belongs to and which `DeviceService` it used to connect to the system.

For details of the design, please see the [document](https://github.com/openyurtio/openyurt/blob/master/docs/proposals/20210310-edge-device-management.md).

## Getting Start

To use the yurt-device-controller, you need to deploy the OpenYurt cluster in advance and meet the following two conditions:

- Deploy the yurt-app-manager, the details of yurt-app-manager are in [here](https://github.com/openyurtio/yurt-app-manager) ；
- Deploy an Edgex Foundry instance in a NodePool by using yurt-edgex-manager, the details of yurt-edgex-manager are in [here](https://github.com/openyurtio/yurt-edgex-manager) .

For a complete example, please check out the [tutorial](docs/yurt-device-controller-tutorial.md)

## Contributing

Contributions are welcome, whether by creating new issues or pull requests. See our [contributing document](https://github.com/openyurtio/openyurt/blob/master/CONTRIBUTING.md) to get started.

## Contact

- Mailing List: openyurt@googlegroups.com
- Slack: [channel](https://join.slack.com/t/openyurt/shared_invite/zt-iw2lvjzm-MxLcBHWm01y1t2fiTD15Gw)
- Dingtalk Group (钉钉讨论群)

<div align="left">
    <img src="https://github.com/openyurtio/openyurt/blob/master/docs/img/ding.jpg" width=25% title="dingtalk">
</div>


## License

Yurt-device-controller is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details. Certain implementations in Yurt-device-controller rely on the existing code from [Kubernetes](https://github.com/kubernetes/kubernetes) and [OpenKruise](https://github.com/openkruise/kruise) the credits go to the original authors.
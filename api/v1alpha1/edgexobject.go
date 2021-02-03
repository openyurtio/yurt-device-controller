package v1alpha1

type EdgeXObject interface {
	IsAddedToEdgeX() bool
}

func (vd *ValueDescriptor) IsAddedToEdgeX() bool {
	return vd.Status.AddedToEdgeX
}

func (dp *DeviceProfile) IsAddedToEdgeX() bool {
	return dp.Status.AddedToEdgeX
}

func (d *Device) IsAddedToEdgeX() bool {
	return d.Status.AddedToEdgeX
}

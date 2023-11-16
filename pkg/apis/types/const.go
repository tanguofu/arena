package types

const (
	CPUResourceName = "cpu"
)

const (
	// defines the nvidia resource name
	NvidiaGPUResourceName = "nvidia.com/gpu"
)

const (
	GPUShareResourceName        = "nvidia.com/gpu"
	GPUCoreShareResourceName    = "tke.cloud.tencent.com/qgpu-core"
	GPUShareCountName           = "tke.cloud.tencent.com/gpu.count"
	GPUShareEnvGPUID            = "ALIYUN_COM_GPU_MEM_IDX"
	GPUShareAllocationLabel     = "scheduler.framework.gpushare.allocation"
	GPUCoreShareAllocationLabel = "gpushare.alibabacloud.com/core-percentage"
	GPUShareNodeLabels          = "qgpu-device-enable=enable"
)

const (
	AliyunGPUResourceName      = "nvidia.com/gpu"
	GPUTopologyAllocationLabel = "topology.kubernetes.io/gpu-group"
	GPUTopologyVisibleGPULabel = "topology.kubernetes.io/gpu-visible"
	GPUTopologyNodeLabels      = "ack.node.gpu.schedule=topology"
)

const (
	MultiTenantIsolationLabel = "arena.kubeflow.org/isolate-user"
	UserNameIdLabel           = "arena.kubeflow.org/uid"
	SSHSecretName             = "arena.kubeflow.org/ssh-secret"
)

// anno
const (
	GpuTypeAnno             = "ti.cloud.tencent.com/gpu-type"
	InstanceTypeAnno        = "ti.cloud.tencent.com/instance-type"
	ResourceGroupIdAnno     = "ti.cloud.tencent.com/resourcegroup-id"
	ResourceGroupRegionAnno = "ti.cloud.tencent.com/region-id"
	EnableRDMAAnno          = "ti.cloud.tencent.com/rdma-vhca"
)

// label
const (
	UserNameNameLabel = "ti.cloud.tencent.com/user-id"
	TaskRegionLabel   = "ti.cloud.tencent.com/region"
)

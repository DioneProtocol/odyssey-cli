// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package constants

import (
	"time"
)

const (
	DefaultPerms755        = 0o755
	WriteReadReadPerms     = 0o644
	WriteReadUserOnlyPerms = 0o600

	BaseDirName = ".odyssey-cli"
	LogDir      = "logs"

	ServerRunFile    = "gRPCserver.run"
	OdysseyCliBinDir = "bin"
	RunDir           = "runs"

	SuffixSeparator              = "_"
	SidecarFileName              = "sidecar.json"
	GenesisFileName              = "genesis.json"
	ElasticSubnetConfigFileName  = "elastic_subnet_config.json"
	SidecarSuffix                = SuffixSeparator + SidecarFileName
	GenesisSuffix                = SuffixSeparator + GenesisFileName
	NodeFileName                 = "node.json"
	NodeCloudConfigFileName      = "node_cloud_config.json"
	AnsibleDir                   = "ansible"
	AnsibleHostInventoryFileName = "hosts"
	StopAWSNode                  = "stop-aws-node"
	CreateAWSNode                = "create-aws-node"
	GetAWSNodeIP                 = "get-aws-node-ip"
	ClustersConfigFileName       = "cluster_config.json"
	ClustersConfigVersion        = "1"
	StakerCertFileName           = "staker.crt"
	StakerKeyFileName            = "staker.key"
	BLSKeyFileName               = "signer.key"
	SidecarVersion               = "1.4.0"

	MaxLogFileSize   = 4
	MaxNumOfLogFiles = 5
	RetainOldFiles   = 0 // retain all old log files

	CloudOperationTimeout = 2 * time.Minute

	ONRRequestTimeout   = 3 * time.Minute
	APIRequestTimeout   = 30 * time.Second
	FastGRPCDialTimeout = 100 * time.Millisecond

	SSHServerStartTimeout = 1 * time.Minute
	SSHScriptTimeout      = 2 * time.Minute
	SSHDirOpsTimeout      = 10 * time.Second
	SSHFileOpsTimeout     = 30 * time.Second
	SSHPOSTTimeout        = 10 * time.Second
	SSHSleepBetweenChecks = 1 * time.Second
	SSHScriptLogFilter    = "_OdysseyCLI_LOG_"
	SSHShell              = "/bin/bash"

	SimulatePublicNetwork = "SIMULATE_PUBLIC_NETWORK"

	TestnetAPIEndpoint = "https://testnode.dioneprotocol.com"
	MainnetAPIEndpoint = "https://testnode.dioneprotocol.com"

	// this depends on bootstrap snapshot
	LocalAPIEndpoint = "http://127.0.0.1:9650"
	LocalNetworkID   = 1337

	DevnetAPIEndpoint = ""
	DevnetNetworkID   = 1338

	DefaultTokenName = "TEST"

	HealthCheckInterval = 100 * time.Millisecond

	// it's unlikely anyone would want to name a snapshot `default`
	// but let's add some more entropy
	SnapshotsDirName = "snapshots"

	DefaultSnapshotName = "default-1654102509"

	BootstrapSnapshotRawBranch = "https://github.com/DioneProtocol/odyssey-cli/raw/develop/"

	CurrentBootstrapNamePath = "currentBootstrapName.txt"

	BootstrapSnapshotArchiveName = "bootstrapSnapshot.tar.gz"
	BootstrapSnapshotLocalPath   = "assets/" + BootstrapSnapshotArchiveName
	BootstrapSnapshotURL         = BootstrapSnapshotRawBranch + BootstrapSnapshotLocalPath
	BootstrapSnapshotSHA256URL   = BootstrapSnapshotRawBranch + "assets/sha256sum.txt"

	BootstrapSnapshotSingleNodeArchiveName = "bootstrapSnapshotSingleNode.tar.gz"
	BootstrapSnapshotSingleNodeLocalPath   = "assets/" + BootstrapSnapshotSingleNodeArchiveName
	BootstrapSnapshotSingleNodeURL         = BootstrapSnapshotRawBranch + BootstrapSnapshotSingleNodeLocalPath
	BootstrapSnapshotSingleNodeSHA256URL   = BootstrapSnapshotRawBranch + "assets/sha256sumSingleNode.txt"

	CliInstallationURL      = "https://raw.githubusercontent.com/DioneProtocol/odyssey-cli/main/scripts/install.sh"
	ExpectedCliInstallErr   = "resource temporarily unavailable"
	EIPLimitErr             = "AddressLimitExceeded"
	ErrCreatingAWSNode      = "failed to create AWS Node"
	ErrCreatingGCPNode      = "failed to create GCP Node"
	ErrReleasingGCPStaticIP = "failed to release gcp static ip"
	KeyDir                  = "key"
	KeySuffix               = ".pk"
	YAMLSuffix              = ".yml"

	Enable = "enable"

	Disable = "disable"

	TimeParseLayout             = "2006-01-02 15:04:05"
	MinStakeWeight              = 1
	DefaultStakeWeight          = 20
	DIONESymbol                 = "DIONE"
	DefaultTestnetStakeDuration = "48h"
	DefaultMainnetStakeDuration = "336h"
	// The absolute minimum is 25 seconds, but set to 1 minute to allow for
	// time to go through the command
	DevnetStakingStartLeadTime                   = 30 * time.Second
	StakingStartLeadTime                         = 5 * time.Minute
	StakingMinimumLeadTime                       = 25 * time.Second
	PrimaryNetworkValidatingStartLeadTimeNodeCmd = 20 * time.Second
	PrimaryNetworkValidatingStartLeadTime        = 1 * time.Minute
	AWSCloudServerRunningState                   = "running"
	OdysseyCLISuffix                             = "-odyssey-cli"
	AWSDefaultCredential                         = "default"
	GCPDefaultImageProvider                      = "ubuntu-os-cloud"
	GCPImageFilter                               = "family=ubuntu-2004* AND architecture=x86_64"
	GCPEnvVar                                    = "GOOGLE_APPLICATION_CREDENTIALS"
	GCPDefaultAuthKeyPath                        = "~/.config/gcloud/application_default_credentials.json"
	CertSuffix                                   = "-kp.pem"
	AWSSecurityGroupSuffix                       = "-sg"
	ExportSubnetSuffix                           = "-export.dat"
	SSHTCPPort                                   = 22
	OdysseygoAPIPort                             = 9650
	OdysseygoP2PPort                             = 9651
	OdysseygoGrafanaPort                         = 3000
	CloudServerStorageSize                       = 1000
	MonitoringCloudServerStorageSize             = 50
	OutboundPort                                 = 0
	SetupCLIFromSourceBranch                     = "main"
	// Set this one to true while testing changes that alter CLI execution on cloud nodes
	// Disable it for releases to save cluster creation time
	EnableSetupCLIFromSource           = false
	BuildEnvGolangVersion              = "1.21.1"
	IsHealthyJSONFile                  = "isHealthy.json"
	IsBootstrappedJSONFile             = "isBootstrapped.json"
	OdysseyGoVersionJSONFile           = "odysseyGoVersion.json"
	SubnetSyncJSONFile                 = "isSubnetSynced.json"
	AnsibleInventoryDir                = "inventories"
	AnsibleTempInventoryDir            = "temp_inventories"
	AnsibleStatusDir                   = "status"
	AnsibleInventoryFlag               = "-i"
	AnsibleExtraArgsIdentitiesOnlyFlag = "--ssh-extra-args='-o IdentitiesOnly=yes'"
	AnsibleSSHShellParams              = "-o IdentitiesOnly=yes -o StrictHostKeyChecking=no"
	AnsibleSSHUseAgentParams           = "-o StrictHostKeyChecking=no"
	AnsibleExtraVarsFlag               = "--extra-vars"

	ConfigOPMCredentialsFileKey  = "credentials-file"
	ConfigOPMAdminAPIEndpointKey = "admin-api-endpoint"
	ConfigNodeConfigKey          = "node-config"
	ConfigMetricsEnabledKey      = "MetricsEnabled"
	ConfigAutorizeCloudAccessKey = "AutorizeCloudAccess"
	OldConfigFileName            = ".odyssey-cli.json"
	OldMetricsConfigFileName     = ".odyssey-cli/config"
	DefaultConfigFileName        = ".odyssey-cli/config.json"

	AWSCloudService              = "Amazon Web Services"
	GCPCloudService              = "Google Cloud Platform"
	AWSDefaultInstanceType       = "c5.2xlarge"
	GCPDefaultInstanceType       = "e2-standard-8"
	AnsibleSSHUser               = "ubuntu"
	AWSNodeAnsiblePrefix         = "aws_node"
	GCPNodeAnsiblePrefix         = "gcp_node"
	CustomVMDir                  = "vms"
	GCPStaticIPPrefix            = "static-ip"
	DioneProtocolOrg             = "DioneProtocol"
	OdysseyGoRepoName            = "odysseygo"
	SubnetEVMRepoName            = "subnet-evm"
	CliRepoName                  = "odyssey-cli"
	SubnetEVMReleaseURL          = "https://github.com/DioneProtocol/subnet-evm/releases/download/%s/%s"
	SubnetEVMArchive             = "subnet-evm_%s_linux_amd64.tar.gz"
	CloudNodeConfigBasePath      = "/home/ubuntu/.odysseygo/"
	CloudNodeSubnetEvmBinaryPath = "/home/ubuntu/.odysseygo/plugins/%s"
	CloudNodeStakingPath         = "/home/ubuntu/.odysseygo/staking/"
	CloudNodeConfigPath          = "/home/ubuntu/.odysseygo/configs/"
	CloudNodeCLIConfigBasePath   = "/home/ubuntu/.odyssey-cli/"
	OdysseygoMonitoringPort      = 9090
	OdysseygoMachineMetricsPort  = 9100
	MonitoringScriptFile         = "monitoring-separate-installer.sh"
	MonitoringDir                = "monitoring"
	DashboardsDir                = "dashboards"
	NodeConfigJSONFile           = "node.json"
	IPAddressSuffix              = "/32"
	OdysseyGoInstallDir          = "odysseygo"
	SubnetEVMInstallDir          = "subnet-evm"

	SubnetEVMBin = "subnet-evm"

	DefaultNodeRunURL = "http://127.0.0.1:9650"

	OPMDir                      = ".opm"
	OPMLogName                  = "opm.log"
	DefaultDioneProtocolPackage = "dioneprotocol/odyssey-plugins-core"
	OPMPluginDir                = "opm_plugins"

	// #nosec G101
	GithubAPITokenEnvVarName = "ODYSSEY_CLI_GITHUB_TOKEN"

	ReposDir                   = "repos"
	SubnetDir                  = "subnets"
	NodesDir                   = "nodes"
	VMDir                      = "vms"
	ChainConfigDir             = "chains"
	AVMKeyName                 = "alpha"
	EVMKeyName                 = "evm"
	PlatformKeyName            = "omega"
	SubnetType                 = "subnet type"
	PrecompileType             = "precompile type"
	CustomAirdrop              = "custom-airdrop"
	NumberOfAirdrops           = "airdrop-addresses"
	SubnetConfigFileName       = "subnet.json"
	ChainConfigFileName        = "chain.json"
	PerNodeChainConfigFileName = "per-node-chain.json"
	NodeConfigFileName         = "node-config.json"

	GitRepoCommitName        = "Odyssey-CLI"
	GitRepoCommitEmail       = "info@dione.network"
	DioneProtocolMaintainers = "DioneProtocol"

	UpgradeBytesFileName      = "upgrade.json"
	UpgradeBytesLockExtension = ".lock"
	NotAvailableLabel         = "Not available"
	BackendCmd                = "odyssey-cli-backend"

	OdysseyGoVersionUnknown            = "n/a"
	OdysseyGoCompatibilityVersionAdded = "v1.9.2"
	OdysseyGoCompatibilityURL          = "https://raw.githubusercontent.com/DioneProtocol/odysseygo/develop/version/compatibility.json"
	SubnetEVMRPCCompatibilityURL       = "https://raw.githubusercontent.com/DioneProtocol/subnet-evm/develop/compatibility.json"

	YesLabel = "Yes"
	NoLabel  = "No"

	SubnetIDLabel     = "SubnetID: "
	BlockchainIDLabel = "BlockchainID: "

	PluginDir = "plugins"

	Network        = "network"
	MultiSig       = "multi-sig"
	SkipUpdateFlag = "skip-update-check"
	LastFileName   = ".last_actions.json"

	DefaultWalletCreationTimeout = 5 * time.Second

	DefaultConfirmTxTimeout = 20 * time.Second

	PayTxsFeesMsg = "pay transaction fees"
)

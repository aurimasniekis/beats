package cpu

import (
	ipg "github.com/aurimasniekis/go-intel-power-gadget"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"strconv"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("intel_power_gadget", "cpu", New)

	ipg.Initialize()
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	intelPackages map[int]*ipg.IntelPowerGadgetPackage
	sampleIds map[int]ipg.SampleId
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The intel_power_gadget cpu metricset is beta.")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	intelPackages := ipg.GetPackages()
	sampleIds := make(map[int]ipg.SampleId)
	for num, pkg := range intelPackages {
		sampleIds[num] = ipg.StartSampling(pkg)
	}

	return &MetricSet{
		BaseMetricSet: base,
		intelPackages: intelPackages,
		sampleIds: sampleIds,
	}, nil
}

func (m *MetricSet) Close() error {
	ipg.Shutdown()

	return nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	for pkgNum, pkg := range m.intelPackages {
		sample := ipg.FinishSampling(m.sampleIds[pkgNum], pkg)

		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"name": "CPU" + strconv.Itoa(pkgNum),
				"package_no": pkgNum,
				"package_cores": sample.Pkg.PackageCores,
				"ia_base_frequency": sample.Pkg.IaBaseFrequency/1000,
				"ia_max_frequency": sample.Pkg.IaMaxFrequency/1000,
				"gt_max_frequency": sample.Pkg.GtMaxFrequency/1000,
				"package_tdp": sample.Pkg.PackageTDP,
				"max_temperature": sample.Pkg.MaxTemperature,
				"ia_frequency": sample.IaFrequency.Mean/1000,
				"ia_frequency_request": sample.IaFrequencyRequest.Mean/1000,
				"ia_power": sample.IaPower.Watts,
				"ia_temperature": sample.IaTemperature.Mean,
				"ia_utilization": sample.IaUtilization,
				"gt_frequency": sample.GtFrequency/1000,
				"gt_frequency_request": sample.GtFrequencyRequest/1000,
				"gt_utilization": sample.GtUtilization,
				"package_power": sample.PackagePower.Watts,
				"platform_power": sample.PlatformPower.Watts,
				"dram_power": sample.DramPower.Watts,
				"package_temperature": sample.PackageTemperature,
				"tdp": sample.Tdp,
			},
		})

		for i := 0; i < sample.Pkg.PackageCores; i++ {
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"name": "CPU" + strconv.Itoa(pkgNum),
					"package_no": pkgNum,
					"core": i,
					"core_name": "CPU" + strconv.Itoa(pkgNum) + " Core " + strconv.Itoa(i),
					"ia_core_frequency": sample.IaCoreFrequency[i].Mean/1000,
					"ia_core_frequency_request": sample.IaCoreFrequencyRequest[i].Mean/1000,
					"ia_core_temperature": sample.IaCoreTemperature[i].Mean,
					"ia_core_utilization": sample.IaCoreUtilization[i],
				},
			})
		}
	}

	for num, pkg := range m.intelPackages {
		m.sampleIds[num] = ipg.StartSampling(pkg)
	}

	return nil
}

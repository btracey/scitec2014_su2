package main

import (
	//"github.com/btracey/su2tools/config"
	"github.com/btracey/su2tools/config/common"
	"github.com/btracey/su2tools/driver"
	//"github.com/btracey/su2tools/nondimensionalize"

	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

var gopath string

func init() {
	gopath = os.Getenv("GOPATH")
	if gopath == "" {
		panic("gopath not set")
	}
}

func Basecase(baseDataLoc, baseoptions string) (*driver.Driver, error) {
	basecase := &driver.Driver{}
	basecase.Name = "base"
	// set options and options list
	err := basecase.SetRelativeOptions(baseoptions, false, nil)
	if err != nil {
		return nil, err
	}
	basecase.Config = "config.cfg"
	basecase.Wd = filepath.Join(baseDataLoc, basecase.Name)

	basecase.Stdout = basecase.Name + "_log.txt"
	basecase.Options.MeshFilename = filepath.Join(baseDataLoc, basecase.Options.MeshFilename)
	return basecase, nil
}

// newBaseDriver gets a new driver with the config options of the base config file,
// with the results directory set by the kind and the subkind, and with the log
// file set
func newBaseDriver(kind, subkind, baseoptions, resultspath string) *driver.Driver {
	drive := &driver.Driver{}
	drive.Name = string(subkind)
	drive.SetRelativeOptions(baseoptions, false, nil)
	drive.Config = subkind + "config.cfg"
	drive.Wd = filepath.Join(resultspath, kind, subkind)
	drive.Stdout = subkind + "_log.txt"
	return drive
}

func AdditionalDrivers(vary string, baseoptions, resultspath string) []*driver.Driver {
	drivers := make([]*driver.Driver, 0)
	switch vary {
	case "ConvNum":
		newConvOptions := []common.Enum{"ROE-2ND_ORDER", "AUSM-2ND_ORDER", "HLLC-2ND_ORDER", "ROE_TURKEL_2ND"}
		for _, name := range newConvOptions {
			drive := newBaseDriver(vary, string(name), baseoptions, resultspath)
			drive.Options.ConvNumMethodFlow = name
			drive.OptionList["ConvNumMethodFlow"] = true
			drivers = append(drivers, drive)
		}
	case "MeanLimiterCoeff":
		newLimiterValues := []float64{0.1, 0.3, 0.5}
		for _, val := range newLimiterValues {
			subkind := "limitcoeff" + strconv.FormatFloat(val, 'g', 4, 64)
			drive := newBaseDriver(vary, subkind, baseoptions, resultspath)
			drive.Options.LimiterCoeff = val
			drive.OptionList["LimiterCoeff"] = true
			drivers = append(drivers, drive)
		}
	case "AdCoeffFlow":
		adcoeff := []float64{0.02, 0.01, 0.005, 0.001}
		for _, val := range adcoeff {
			subkind := "adcoeff_" + strconv.FormatFloat(val, 'g', 4, 64)
			drive := newBaseDriver(vary, subkind, baseoptions, resultspath)
			//drive.Options
			drive.Options.LimiterCoeff = val
			drive.OptionList["LimiterCoeff"] = true
			drivers = append(drivers, drive)
		}
	case "TurbOrder":
		drive := newBaseDriver(vary, "firstorder", baseoptions, resultspath)
		drive.Options.ConvNumMethodTurb = "Scalar_Upwind-1st_Order"
		drive.OptionList["ConvNumMethodTurb"] = true
		drivers = append(drivers, drive)
	case "ViscNumMethod":
		drive := newBaseDriver(vary, "avggrad", baseoptions, resultspath)
		drive.Options.ViscNumMethodFlow = "AVG_GRAD"
		drive.OptionList["ViscNumMethodFlow"] = true
		drivers = append(drivers, drive)
		drive = newBaseDriver(vary, "avggrad_corr", baseoptions, resultspath)
		drive.Options.ViscNumMethodFlow = "AVG_GRAD_CORRECTED"
		drive.OptionList["ViscNumMethodFlow"] = true
		drivers = append(drivers, drive)
	case "TurbNumMethod":
		drive := newBaseDriver(vary, "avggrad", baseoptions, resultspath)
		drive.Options.ViscNumMethodTurb = "AVG_GRAD"
		drive.OptionList["ViscNumMethodFlow"] = true
		drivers = append(drivers, drive)
		drive = newBaseDriver(vary, "avggrad_corr", baseoptions, resultspath)
		drive.Options.ViscNumMethodTurb = "AVG_GRAD_CORRECTED"
		drive.OptionList["ViscNumMethodFlow"] = true
		drivers = append(drivers, drive)
	case "none":
	default:
		panic("Unrecognized vary " + vary)
	}
	return drivers
}

func main() {
	// Set up the flags and parse them
	var vary string
	flag.StringVar(&vary, "vary", "none", "Which variable should be varied from the baseline")
	flag.Parse()

	// Set up the location for the base config file and the meshes
	baseLoc := filepath.Join(gopath, "data", "scitec_2014", "su2paper_sst")
	baseConfig := filepath.Join(baseLoc, "base_flatplate_config.cfg")

	// Initialize variables
	var drivers []*driver.Driver

	// Set up the driver for the baseline flow
	basecase, err := Basecase(baseLoc, baseConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	drivers = append(drivers, basecase)

	resultLoc := filepath.Join(gopath, "results", "scitec_2014", "su2paper_sst")
	// Collect the extra drivers for how we are running the case
	extraDrivers := AdditionalDrivers(vary, baseConfig, resultLoc)

	// Append the extra drivers
	drivers = append(drivers, extraDrivers...)

	// Set up how we want to run the cases (all at a time but in parallel? One at a time but each in parallel?)
	//su2caller := driver.Serial{}
	//su2caller.Concurrent = true
	su2caller := driver.Parallel{}
	su2caller.Concurrent = false
	su2caller.NumCores = 4
	fmt.Println("Num drivers = ", len(drivers))

	// Run all of the cases
	fmt.Println("Running case ", vary)
	errors := driver.RunCases(drivers, su2caller, true)
	// Ouput the success or failure of
	for i := range drivers {
		if errors == nil && errors[i] == nil {
			fmt.Println("Error running case " + drivers[i].Name + "_" + errors[i].Error())
		} else {
			fmt.Println("Case " + drivers[i].Name + " ran successfully")
		}
		// Copy restart to solution to mark that the case has been run
		drivers[i].CopyRestartToSolution()
	}
}

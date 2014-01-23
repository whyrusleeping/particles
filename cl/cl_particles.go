package main

import (
	"fmt"
	"github.com/tones111/go-opencl/cl"
	"github.com/tones111/raw"
)

type CLSetup struct {
	kernel *cl.Kernel
	queue *cl.CommandQueue
	context *cl.Context
	buffer *cl.Buffer
	size uint32
}

func StartupCL(p []Particle) *CLSetup {
	bytes := uint32(len(raw.ByteSlice(p)))
	for _,platform := range cl.GetPlatforms() {
		for _,dev := range platform.Devices {
			//Get the device were going to run on.
			fmt.Println("  Platform Name:", platform.Property(cl.PLATFORM_NAME))
			context, err := cl.NewContextOfDevices(map[cl.ContextParameter]interface{}{cl.CONTEXT_PLATFORM: platform}, []cl.Device{dev})
			if err != nil {
				panic(err)
			}

			//Create the queue we will use to communicate with the device
			queue, err := context.NewCommandQueue(dev, cl.QUEUE_NIL)
			if err != nil {
				panic(err)
			}

			//Load the program
			program, err := context.NewProgramFromFile("gravity.cl")
			if err != nil {
				panic(err)
			}

			//Try building the program
			if err := program.Build(nil, ""); err != nil {
				if status := program.BuildStatus(dev); status != cl.BUILD_SUCCESS {
					panic(fmt.Sprintf("Build Error:\n%s\n", program.Property(dev, cl.BUILD_LOG)))
				}
				panic(err)
			}

			//Build the kernel
			kernel, err := program.NewKernelNamed("updatepart")
			if err != nil {
				panic(err)
			}

			buffer,err := context.NewBuffer(cl.MEM_READ_WRITE, bytes)
			if err != nil {
				panic(err)
			}

			cls := new(CLSetup)
			cls.context = context
			cls.kernel = kernel
			cls.queue = queue
			cls.size = bytes
			cls.buffer = buffer

			return cls
		}
	}
	return nil
}

func (c *CLSetup) Execute(p []Particle) {
	err := c.queue.EnqueueWriteBuffer(c.buffer, raw.ByteSlice(p), 0)
	if err != nil {
		panic(err)
	}

	err = c.kernel.SetArgs(0, []interface{}{c.buffer})
	if err != nil {
		panic(err)
	}

	err = c.queue.EnqueueKernel(c.kernel, []cl.Size{0}, []cl.Size{cl.Size(len(p))}, []cl.Size{1})
	if err != nil {
		panic(err)
	}

	out, err := c.queue.EnqueueReadBuffer(c.buffer, 0, c.size)
	if err != nil {
		panic(err)
	}

	raw.ByteCopy(p, out)
}

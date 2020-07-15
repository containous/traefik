package ecs

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/cenkalti/backoff/v4"
	"github.com/patrickmn/go-cache"

	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/containous/traefik/v2/pkg/job"
	"github.com/containous/traefik/v2/pkg/log"
	"github.com/containous/traefik/v2/pkg/provider"
	"github.com/containous/traefik/v2/pkg/safe"
)

// Provider holds configurations of the provider.
type Provider struct {
	Constraints      string `description:"Constraints is an expression that Traefik matches against the container's labels to determine whether to create any route for that container." json:"constraints,omitempty" toml:"constraints,omitempty" yaml:"constraints,omitempty" export:"true"`
	ExposedByDefault bool   `description:"Expose services by default" json:"exposedByDefault,omitempty" toml:"exposedByDefault,omitempty" yaml:"exposedByDefault,omitempty" export:"true"`
	RefreshSeconds   int    `description:"Polling interval (in seconds)" json:"refreshSeconds,omitempty" toml:"refreshSeconds,omitempty" yaml:"refreshSeconds,omitempty" export:"true"`
	DefaultRule      string `description:"Default rule." json:"defaultRule,omitempty" toml:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`

	// Provider lookup parameters.
	Clusters             []string `description:"ECS Clusters name" json:"clusters,omitempty" toml:"clusters,omitempty" yaml:"clusters,omitempty" export:"true"`
	AutoDiscoverClusters bool     `description:"Auto discover cluster" json:"autoDiscoverClusters,omitempty" toml:"autoDiscoverClusters,omitempty" yaml:"autoDiscoverClusters,omitempty" export:"true"`
	Region               string   `description:"The AWS region to use for requests"  json:"region,omitempty" toml:"region,omitempty" yaml:"region,omitempty" export:"true"`
	AccessKeyID          string   `description:"The AWS credentials access key to use for making requests" json:"accessKeyID,omitempty" toml:"accessKeyID,omitempty" yaml:"accessKeyID,omitempty"`
	SecretAccessKey      string   `description:"The AWS credentials access key to use for making requests" json:"secretAccessKey,omitempty" toml:"secretAccessKey,omitempty" yaml:"secretAccessKey,omitempty"`
	defaultRuleTpl       *template.Template
}

type ecsInstance struct {
	Name                string
	ID                  string
	containerDefinition *ecs.ContainerDefinition
	machine             *machine
	Labels              map[string]string
	ExtraConf           configuration
}

type portMapping struct {
	containerPort int64
	hostPort      int64
	protocol      string
}

type machine struct {
	state        string
	privateIP    string
	ports        []portMapping
	healthStatus string
}

type awsClient struct {
	ecs *ecs.ECS
	ec2 *ec2.EC2
}

// DefaultTemplateRule The default template for the default rule.
const DefaultTemplateRule = "Host(`{{ normalize .Name }}`)"

var _ provider.Provider = (*Provider)(nil)
var existingTaskDefCache = cache.New(30*time.Minute, 5*time.Minute)

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {
	p.Clusters = []string{"default"}
	p.AutoDiscoverClusters = false
	p.ExposedByDefault = true
	p.RefreshSeconds = 15
	p.DefaultRule = DefaultTemplateRule
}

// Init the provider.
func (p *Provider) Init() error {
	defaultRuleTpl, err := provider.MakeDefaultRuleTemplate(p.DefaultRule, nil)
	if err != nil {
		return fmt.Errorf("error while parsing default rule: %w", err)
	}

	p.defaultRuleTpl = defaultRuleTpl
	return nil
}

func (p *Provider) createClient(logger log.Logger) (*awsClient, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	ec2meta := ec2metadata.New(sess)
	if p.Region == "" {
		logger.Infoln("No EC2 region provided, querying instance metadata endpoint...")
		identity, err := ec2meta.GetInstanceIdentityDocument()
		if err != nil {
			return nil, err
		}
		p.Region = identity.Region
	}

	cfg := &aws.Config{
		Region: &p.Region,
		Credentials: credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.StaticProvider{
					Value: credentials.Value{
						AccessKeyID:     p.AccessKeyID,
						SecretAccessKey: p.SecretAccessKey,
					},
				},
				&credentials.EnvProvider{},
				&credentials.SharedCredentialsProvider{},
				defaults.RemoteCredProvider(*(defaults.Config()), defaults.Handlers()),
			}),
	}

	cfg.WithLogger(aws.LoggerFunc(func(args ...interface{}) {
		logger.Debug(args...)
	}))

	return &awsClient{
		ecs.New(sess, cfg),
		ec2.New(sess, cfg),
	}, nil
}

// Provide configuration to traefik from ECS.
func (p Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	pool.GoCtx(func(routineCtx context.Context) {
		ctxLog := log.With(routineCtx, log.Str(log.ProviderName, "ecs"))
		logger := log.FromContext(ctxLog)

		operation := func() error {
			awsClient, err := p.createClient(logger)
			if err != nil {
				return err
			}

			configuration, err := p.loadECSConfig(ctxLog, awsClient)
			if err != nil {
				return err
			}

			configurationChan <- dynamic.Message{
				ProviderName:  "ecs",
				Configuration: configuration,
			}

			reload := time.NewTicker(time.Second * time.Duration(p.RefreshSeconds))
			defer reload.Stop()

			for {
				select {
				case <-reload.C:
					configuration, err := p.loadECSConfig(ctxLog, awsClient)
					if err != nil {
						logger.Errorf("Failed to load ECS configuration, error %s", err)
						return err
					}

					configurationChan <- dynamic.Message{
						ProviderName:  "ecs",
						Configuration: configuration,
					}
				case <-routineCtx.Done():
					return nil
				}
			}
		}

		notify := func(err error, time time.Duration) {
			logger.Errorf("Provider connection error %+v, retrying in %s", err, time)
		}
		err := backoff.RetryNotify(safe.OperationWithRecover(operation), job.NewBackOff(backoff.NewExponentialBackOff()), notify)
		if err != nil {
			logger.Errorf("Cannot connect to Provider api %+v", err)
		}
	})

	return nil
}

// Find all running Provider tasks in a cluster, also collect the task definitions (for docker labels)
// and the EC2 instance data.
func (p *Provider) listInstances(ctx context.Context, client *awsClient) ([]ecsInstance, error) {
	logger := log.FromContext(ctx)

	var clustersArn []*string
	var clusters []string

	if p.AutoDiscoverClusters {
		input := &ecs.ListClustersInput{}
		for {
			result, err := client.ecs.ListClusters(input)
			if err != nil {
				return nil, err
			}
			if result != nil {
				clustersArn = append(clustersArn, result.ClusterArns...)
				input.NextToken = result.NextToken
				if result.NextToken == nil {
					break
				}
			} else {
				break
			}
		}
		for _, cArn := range clustersArn {
			clusters = append(clusters, *cArn)
		}
	} else {
		clusters = p.Clusters
	}

	var instances []ecsInstance

	logger.Debugf("ECS Clusters: %s", clusters)
	for _, c := range clusters {
		input := &ecs.ListTasksInput{
			Cluster:       &c,
			DesiredStatus: aws.String(ecs.DesiredStatusRunning),
		}

		tasks := make(map[string]*ecs.Task)
		err := client.ecs.ListTasksPagesWithContext(ctx, input, func(page *ecs.ListTasksOutput, lastPage bool) bool {
			if len(page.TaskArns) > 0 {
				resp, err := client.ecs.DescribeTasksWithContext(ctx, &ecs.DescribeTasksInput{
					Tasks:   page.TaskArns,
					Cluster: &c,
				})
				if err != nil {
					logger.Errorf("Unable to describe tasks for %v", page.TaskArns)
				} else {
					for _, t := range resp.Tasks {
						if aws.StringValue(t.LastStatus) == ecs.DesiredStatusRunning {
							tasks[aws.StringValue(t.TaskArn)] = t
						}
					}
				}
			}
			return !lastPage
		})

		if err != nil {
			logger.Error("Unable to list tasks")
			return nil, err
		}

		// Skip to the next cluster if there are no tasks found on
		// this cluster.
		if len(tasks) == 0 {
			continue
		}

		ec2Instances, err := p.lookupEc2Instances(ctx, client, &c, tasks)
		if err != nil {
			return nil, err
		}

		taskDefinitions, err := p.lookupTaskDefinitions(ctx, client, tasks)
		if err != nil {
			return nil, err
		}

		for key, task := range tasks {
			containerInstance := ec2Instances[aws.StringValue(task.ContainerInstanceArn)]
			taskDef := taskDefinitions[key]

			for _, container := range task.Containers {
				var containerDefinition *ecs.ContainerDefinition
				for _, def := range taskDef.ContainerDefinitions {
					if aws.StringValue(container.Name) == aws.StringValue(def.Name) {
						containerDefinition = def
						break
					}
				}

				if containerDefinition == nil {
					logger.Debugf("Unable to find container definition for %s", aws.StringValue(container.Name))
					continue
				}

				var mach *machine
				if len(task.Attachments) != 0 {
					var ports []portMapping
					for _, mapping := range containerDefinition.PortMappings {
						if mapping != nil {
							var protocol = "TCP"
							if aws.StringValue(mapping.Protocol) == "udp" {
								protocol = "UDP"
							}

							ports = append(ports, portMapping{
								hostPort:      aws.Int64Value(mapping.HostPort),
								containerPort: aws.Int64Value(mapping.ContainerPort),
								protocol:      protocol,
							})
						}
					}
					mach = &machine{
						privateIP:    aws.StringValue(container.NetworkInterfaces[0].PrivateIpv4Address),
						ports:        ports,
						state:        aws.StringValue(task.LastStatus),
						healthStatus: aws.StringValue(task.HealthStatus),
					}
				} else {
					if containerInstance == nil {
						logger.Errorf("Unable to find container instance information for %s", aws.StringValue(container.Name))
						continue
					}

					var ports []portMapping
					for _, mapping := range container.NetworkBindings {
						if mapping != nil {
							ports = append(ports, portMapping{
								hostPort:      aws.Int64Value(mapping.HostPort),
								containerPort: aws.Int64Value(mapping.ContainerPort),
							})
						}
					}
					mach = &machine{
						privateIP: aws.StringValue(containerInstance.PrivateIpAddress),
						ports:     ports,
						state:     aws.StringValue(containerInstance.State.Name),
					}
				}

				instance := ecsInstance{
					Name:                fmt.Sprintf("%s-%s", strings.Replace(aws.StringValue(task.Group), ":", "-", 1), *container.Name),
					ID:                  key[len(key)-12:],
					containerDefinition: containerDefinition,
					machine:             mach,
					Labels:              aws.StringValueMap(containerDefinition.DockerLabels),
				}

				extraConf, err := p.getConfiguration(instance)
				if err != nil {
					log.FromContext(ctx).Errorf("Skip container %s: %w", getServiceName(instance), err)
					continue
				}
				instance.ExtraConf = extraConf

				instances = append(instances, instance)
			}
		}
	}

	return instances, nil
}

func (p *Provider) loadECSConfig(ctx context.Context, client *awsClient) (*dynamic.Configuration, error) {
	instances, err := p.listInstances(ctx, client)
	if err != nil {
		return nil, err
	}

	return p.buildConfiguration(ctx, instances), nil
}

func (p *Provider) lookupEc2Instances(ctx context.Context, client *awsClient, clusterName *string, ecsDatas map[string]*ecs.Task) (map[string]*ec2.Instance, error) {
	logger := log.FromContext(ctx)
	instanceIds := make(map[string]string)
	ec2Instances := make(map[string]*ec2.Instance)

	var containerInstancesArns []*string
	var instanceArns []*string

	for _, task := range ecsDatas {
		if task.ContainerInstanceArn != nil {
			containerInstancesArns = append(containerInstancesArns, task.ContainerInstanceArn)
		}
	}

	for _, arns := range p.chunkIDs(containerInstancesArns) {
		resp, err := client.ecs.DescribeContainerInstancesWithContext(ctx, &ecs.DescribeContainerInstancesInput{
			ContainerInstances: arns,
			Cluster:            clusterName,
		})

		if err != nil {
			logger.Errorf("Unable to describe container instances: %v", err)
			return nil, err
		}

		for _, container := range resp.ContainerInstances {
			instanceIds[aws.StringValue(container.Ec2InstanceId)] = aws.StringValue(container.ContainerInstanceArn)
			instanceArns = append(instanceArns, container.Ec2InstanceId)
		}
	}

	if len(instanceArns) > 0 {
		for _, ids := range p.chunkIDs(instanceArns) {
			input := &ec2.DescribeInstancesInput{
				InstanceIds: ids,
			}

			err := client.ec2.DescribeInstancesPagesWithContext(ctx, input, func(page *ec2.DescribeInstancesOutput, lastPage bool) bool {
				if len(page.Reservations) > 0 {
					for _, r := range page.Reservations {
						for _, i := range r.Instances {
							if i.InstanceId != nil {
								ec2Instances[instanceIds[aws.StringValue(i.InstanceId)]] = i
							}
						}
					}
				}
				return !lastPage
			})

			if err != nil {
				logger.Errorf("Unable to describe instances: %v", err)
				return nil, err
			}
		}
	}

	return ec2Instances, nil
}

func (p *Provider) lookupTaskDefinitions(ctx context.Context, client *awsClient, taskDefArns map[string]*ecs.Task) (map[string]*ecs.TaskDefinition, error) {
	logger := log.FromContext(ctx)
	taskDef := make(map[string]*ecs.TaskDefinition)

	for arn, task := range taskDefArns {
		if definition, ok := existingTaskDefCache.Get(arn); ok {
			taskDef[arn] = definition.(*ecs.TaskDefinition)
			logger.Debugf("Found cached task definition for %s. Skipping the call", arn)
		} else {
			resp, err := client.ecs.DescribeTaskDefinitionWithContext(ctx, &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: task.TaskDefinitionArn,
			})

			if err != nil {
				logger.Errorf("Unable to describe task definition: %v", err)
				return nil, err
			}

			taskDef[arn] = resp.TaskDefinition
			existingTaskDefCache.Set(arn, resp.TaskDefinition, cache.DefaultExpiration)
		}
	}
	return taskDef, nil
}

// chunkIDs ECS expects no more than 100 parameters be passed to a API call;
// thus, pack each string into an array capped at 100 elements.
func (p *Provider) chunkIDs(ids []*string) [][]*string {
	var chuncked [][]*string
	for i := 0; i < len(ids); i += 100 {
		var sliceEnd int
		if i+100 < len(ids) {
			sliceEnd = i + 100
		} else {
			sliceEnd = len(ids)
		}
		chuncked = append(chuncked, ids[i:sliceEnd])
	}
	return chuncked
}

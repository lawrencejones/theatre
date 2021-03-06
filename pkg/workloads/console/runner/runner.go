package runner

import (
	"context"
	"fmt"

	workloadsv1alpha1 "github.com/gocardless/theatre/pkg/apis/workloads/v1alpha1"
	"github.com/gocardless/theatre/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Runner is responsible for managing the lifecycle of a console
type Runner struct {
	kubeClient    kubernetes.Interface
	theatreClient versioned.Interface
}

// Options defines the parameters that can be set upon a new console
type Options struct {
	Cmd     []string
	Timeout int
	Reason  string

	// TODO: For now we assume that all consoles are interactive, i.e. we setup a TTY on
	// them when spawning them. This does not enforce a requirement to attach to the console
	// though.
	// Later on we may need to implement non-interactive consoles, for processes which
	// expect a TTY to not be present?
	// However with these types of consoles it will not be possible to send input to them
	// when reattaching, e.g. attempting to send a SIGINT to cancel the running process.
	// Interactive bool
}

// New builds a runner
func New(coreClient kubernetes.Interface, theatreClient versioned.Interface) *Runner {
	return &Runner{
		kubeClient:    coreClient,
		theatreClient: theatreClient,
	}
}

// Create builds a console according to the supplied options and submits it to the API
func (c *Runner) Create(namespace string, template workloadsv1alpha1.ConsoleTemplate, opts Options) (*workloadsv1alpha1.Console, error) {
	csl := &workloadsv1alpha1.Console{
		ObjectMeta: metav1.ObjectMeta{
			// Let Kubernetes generate a unique name
			GenerateName: template.Name + "-",
			Labels:       labels.Merge(labels.Set{}, template.Labels),
		},
		Spec: workloadsv1alpha1.ConsoleSpec{
			ConsoleTemplateRef: corev1.LocalObjectReference{Name: template.Name},
			// If the flag is not provided then the value will default to 0. The controller
			// should detect this and apply the default timeout that is defined in the template.
			TimeoutSeconds: opts.Timeout,
			Command:        opts.Cmd,
			Reason:         opts.Reason,
		},
	}

	return c.theatreClient.WorkloadsV1alpha1().Consoles(namespace).Create(csl)
}

// FindTemplateBySelector will search for a template matching the given label
// selector and return errors if none or multiple are found (when the selector
// is too broad)
func (c *Runner) FindTemplateBySelector(namespace string, labelSelector string) (*workloadsv1alpha1.ConsoleTemplate, error) {
	client := c.theatreClient.WorkloadsV1alpha1().ConsoleTemplates(namespace)

	templates, err := client.List(
		metav1.ListOptions{
			LabelSelector: labelSelector,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list consoles templates")
	}

	if len(templates.Items) != 1 {
		identifiers := []string{}
		for _, item := range templates.Items {
			identifiers = append(identifiers, item.Namespace+"/"+item.Name)
		}

		return nil, errors.Errorf(
			"expected to discover 1 console template, but actually found: %s",
			identifiers,
		)
	}

	template := templates.Items[0]

	return &template, nil
}

func (c *Runner) FindConsoleByName(namespace, name string) (*workloadsv1alpha1.Console, error) {
	// We must List then filter the slice instead of calling Get(name), otherwise
	// the real Kubernetes client will return the following error when namespace
	// is empty: "an empty namespace may not be set when a resource name is
	// provided".
	// The fake clientset generated by client-gen will not replicate this error in
	// unit tests.
	allConsolesInNamespace, err := c.theatreClient.WorkloadsV1alpha1().Consoles(namespace).
		List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var matchingConsoles []workloadsv1alpha1.Console
	for _, console := range allConsolesInNamespace.Items {
		if console.Name == name {
			matchingConsoles = append(matchingConsoles, console)
		}
	}

	if len(matchingConsoles) == 0 {
		return nil, fmt.Errorf("no consoles found with name: %s", name)
	}
	if len(matchingConsoles) > 1 {
		return nil, fmt.Errorf("too many consoles found with name: %s, please specify namespace", name)
	}

	return &matchingConsoles[0], nil
}

func (c *Runner) ListConsolesByLabelsAndUser(namespace, username, labelSelector string) ([]workloadsv1alpha1.Console, error) {
	// We cannot use a FieldSelector on spec.user in conjunction with the
	// LabelSelector for CRD types like Console. The error message "field label
	// not supported: spec.user" is returned by the real Kubernetes client.
	// See https://github.com/kubernetes/kubernetes/issues/53459.
	csls, err := c.theatreClient.WorkloadsV1alpha1().Consoles(namespace).
		List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}

	var filtered []workloadsv1alpha1.Console
	for _, csl := range csls.Items {
		if username == "" || csl.Spec.User == username {
			filtered = append(filtered, csl)
		}
	}
	return filtered, err
}

// WaitUntilReady will block until the console reaches a phase that indicates
// that it's ready to be attached to, or has failed.
func (c *Runner) WaitUntilReady(ctx context.Context, createdCsl workloadsv1alpha1.Console) (*workloadsv1alpha1.Console, error) {
	isRunning := func(csl *workloadsv1alpha1.Console) bool {
		return csl != nil && csl.Status.Phase == workloadsv1alpha1.ConsoleRunning
	}
	isStopped := func(csl *workloadsv1alpha1.Console) bool {
		return csl != nil && csl.Status.Phase == workloadsv1alpha1.ConsoleStopped
	}

	listOptions := metav1.SingleObject(createdCsl.ObjectMeta)
	client := c.theatreClient.WorkloadsV1alpha1().Consoles(createdCsl.Namespace)

	w, err := client.Watch(listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "error watching console")
	}

	// Get the console, because watch will only give us an event when something
	// is changed, and the phase could have already stabilised before the watch
	// is set up.
	csl, err := client.Get(createdCsl.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "error retrieving console")
	}

	// If the console is already running then there's nothing to do
	if isRunning(csl) {
		return csl, nil
	}
	if isStopped(csl) {
		return nil, fmt.Errorf("console is Stopped")
	}

	status := w.ResultChan()
	defer w.Stop()

	for {
		select {
		case event, ok := <-status:
			// If our channel is closed, exit with error, as we'll otherwise assume
			// we were successful when we never reached this state.
			if !ok {
				return nil, fmt.Errorf("watch channel closed")
			}

			csl = event.Object.(*workloadsv1alpha1.Console)
			if isRunning(csl) {
				return csl, nil
			}
			if isStopped(csl) {
				return nil, fmt.Errorf("console is Stopped")
			}
		case <-ctx.Done():
			if csl == nil {
				return nil, errors.Wrap(ctx.Err(), "console not found")
			}
			return nil, errors.Wrap(ctx.Err(), fmt.Sprintf(
				"console's last phase was: '%v'", csl.Status.Phase),
			)
		}
	}
}

// GetAttachablePod returns an attachable pod for the given console
func (c *Runner) GetAttachablePod(csl *workloadsv1alpha1.Console) (*corev1.Pod, error) {
	pod, err := c.kubeClient.CoreV1().Pods(csl.Namespace).Get(csl.Status.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	for _, c := range pod.Spec.Containers {
		if c.TTY {
			return pod, nil
		}
	}

	return nil, errors.New("no attachable pod found")
}

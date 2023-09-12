package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	secretv1alpha1 "github.com/fr123k/github-operator/api/v1alpha1"
)

var _ = Describe("CronJob controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		CronjobName      = "test-cronjob"
		CronjobNamespace = "default"
		JobName          = "test-job"
		timeout          = time.Second * 30
		duration         = time.Second * 10
		interval         = time.Millisecond * 250
	)

	Context("When updating CronJob Status", func() {
		It("Should increase CronJob Status.Active count when new Jobs are created", func() {
			By("By creating a new CronJob")
			ctx := context.Background()
			secret := &secretv1alpha1.GithubSecret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "secret.fr123k.uk/v1alpha1",
					Kind:       "GithubSecret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      CronjobName,
					Namespace: CronjobNamespace,
				},
				Spec: secretv1alpha1.GithubSecretSpec{
					Repository: "repo",
					DependaBotSecrets: secretv1alpha1.DependaBotSecrets{
						Secrets: []secretv1alpha1.Secrets{
							{Key: "key", Name: "name", Source: "source"},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
			cronjobLookupKey := types.NamespacedName{Name: CronjobName, Namespace: CronjobNamespace}
			createdCronjob := &secretv1alpha1.GithubSecret{}

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, cronjobLookupKey, createdCronjob)
				if err != nil {
					return false
				}
				return len(createdCronjob.Status.Conditions) > 0
			}, timeout, interval).Should(BeTrue())
			// Let's make sure our Schedule string value was properly converted/handled.
			Expect(createdCronjob.Status.Conditions).ShouldNot(BeNil())
			Expect(createdCronjob.Status.Conditions[0].Reason).Should(Equal("Ready"))
			Expect(string(createdCronjob.Status.Conditions[0].Status)).Should(Equal("True"))
		})
	})
})

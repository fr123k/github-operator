/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fr123k/github-operator/api/v1alpha1"
	secretv1alpha1 "github.com/fr123k/github-operator/api/v1alpha1"
	"github.com/fr123k/github-operator/pkg/config"
	"github.com/fr123k/github-operator/pkg/gcloud"
	"github.com/fr123k/github-operator/pkg/github"
	"github.com/go-logr/logr"
)

var GithubSecretOperatorNamespace string

const (
	Finalizer = "secret.fr123k.uk/finalizer"
)

// GithubSecretReconciler reconciles a GithubSecret object
type GithubSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Github github.GithubClient
	GCloud gcloud.GCloudClient

	Config config.Config
}

//+kubebuilder:rbac:groups=secret.fr123k.uk,resources=githubsecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=secret.fr123k.uk,resources=githubsecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=secret.fr123k.uk,resources=githubsecrets/finalizers,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *GithubSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)

	reqLogger.Info("Reconciling GithubSecret")

	// Fetch the Github Secret Operator Secret instance
	instance := &secretv1alpha1.GithubSecret{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile req.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the req.
		reqLogger.Error(err, "Reconcile", "Secret", instance)
		return reconcile.Result{}, err
	}

	if instance.Status.Conditions != nil {
		for _, v := range instance.Status.Conditions {
			if v.Type == secretv1alpha1.ConditionTypeReady {
				return reconcile.Result{}, nil
			}
		}
	}

	repository := instance.Spec.Repository

	secretsConfig := map[string]v1alpha1.Secrets{}
	for _, v := range instance.Spec.DependaBotSecrets.Secrets {
		secretsConfig[v.Name] = v
	}

	secrets, err := r.Github.ListDependaBotSecrets(repository)
	if err != nil {
		msg := fmt.Sprintf("failed to list DependaBot secrets. Error:%s", err.Error())
		reqLogger.Error(err, msg, "Secret", instance)
		apimeta.SetStatusCondition(&instance.Status.Conditions, FailedCondition(msg, secretv1alpha1.ConditionTypeGithubActionSecretError, instance.GetGeneration()))
		return reconcile.Result{}, err
	}

	for _, secret := range secrets.Secrets {
		delete(secretsConfig, secret.Name)
	}

	for _, secret := range secretsConfig {
		value, err := r.GCloud.GetSecretValue(secret.Key)
		if err != nil {
			msg := fmt.Sprintf("Error:%s", err.Error())
			reqLogger.Error(err, msg)
			apimeta.SetStatusCondition(&instance.Status.Conditions, FailedCondition(msg, secretv1alpha1.ConditionTypeGCPSecretManagerError, instance.GetGeneration()))
			err = r.Status().Update(ctx, instance)
			if err != nil {
				log.Error(err, "Failed to update GithubSecrets status")
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Minute}, err
		}

		secret, err := r.Github.AddDependaBotSecrets(r.Config.Owner, repository, secret.Name, *value)
		if err != nil {
			msg := fmt.Sprintf("Error:%s", err.Error())
			reqLogger.Error(err, msg)
			apimeta.SetStatusCondition(&instance.Status.Conditions, FailedCondition(msg, secretv1alpha1.ConditionTypeGithubActionSecretError, instance.GetGeneration()))
		} else {
			reqLogger.Info("added secret", "secret", secret.Name, "repository", repository)
		}
	}

	reqLogger.Info("Reconcile GithubSecret", "GithubSecrets", instance.Spec)

	apimeta.SetStatusCondition(&instance.Status.Conditions, Condition(metav1.ConditionTrue, fmt.Sprintf("Secret %s in ready state", instance.Name), secretv1alpha1.ConditionTypeReady, instance.GetGeneration()))

	err = r.Status().Update(ctx, instance)
	if err != nil {
		log.Error(err, "Failed to update GithubSecrets status")
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// TODO would remove any Github Action DependaBot secret if the CR is deleted
//
//lint:ignore U1000 Ignore could be used in the future to cleanup secrets
func (r *GithubSecretReconciler) finalize(ctx context.Context, log logr.Logger, instance *secretv1alpha1.GithubSecret) error {
	ke := client.ObjectKey{
		Name:      "github-secret-operator",
		Namespace: GithubSecretOperatorNamespace,
	}

	akSecret := v1.Secret{}
	err := r.Client.Get(ctx, ke, &akSecret)
	if err != nil {
		return err
	}

	log.Info("Read Configuration Github Secret Operator", "Secret", akSecret)

	for k, v := range akSecret.Data {
		os.Setenv(k, string(v))
	}

	cfg, ctx := config.Configure()

	gh := github.NewClient(cfg, github.WithContext(ctx))
	for _, v := range instance.Spec.DependaBotSecrets.Secrets {
		err := gh.RemoveDependaBotSecrets(instance.Spec.Repository, v.Name)
		if err != nil {
			log.Error(err, "Remove DependaBot Secrect", "Repo", instance.Spec.Repository, "Secret", v.Name)
		}
	}
	log.Info("Successfully removed Github Secrets")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretv1alpha1.GithubSecret{}).
		// WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		Complete(r)
}

func Condition(status metav1.ConditionStatus, msg string, reason string, generation int64) metav1.Condition {
	return metav1.Condition{
		Status:             status,
		Reason:             reason,
		Message:            msg,
		Type:               reason,
		ObservedGeneration: generation,
	}
}

func FailedCondition(msg string, reason string, generation int64) metav1.Condition {
	return Condition(metav1.ConditionFalse, msg, reason, generation)
}

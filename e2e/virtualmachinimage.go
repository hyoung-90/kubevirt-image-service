package e2e

import (
	"context"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"kubevirt-image-service/pkg/apis"
	"kubevirt-image-service/pkg/apis/hypercloud/v1alpha1"
	"testing"
	"time"
)

func virtualMachinImageTest(t *testing.T, ctx *framework.Context, cleanupOptions *framework.CleanupOptions, retryInterval, timeout time.Duration) {
	ns, err := ctx.GetWatchNamespace()
	if err != nil {
		t.Fatal(err)
	}

	sc := &v1alpha1.VirtualMachineImage{}
	err = framework.AddToFrameworkScheme(apis.AddToScheme, sc)
	if err != nil {
		t.Fatal(err)
	}

	storageClassName := "rook-ceph-block"
	vmi := &v1alpha1.VirtualMachineImage{
		TypeMeta: v1.TypeMeta{
			Kind:       "VirtualMachineImage",
			APIVersion: "hypercloud.tmaxanc.com/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "testcr",
			Namespace: ns,
		},
		Spec: v1alpha1.VirtualMachineImageSpec{
			Source: v1alpha1.VirtualMachineImageSource{
				HTTP: "https://download.cirros-cloud.net/contrib/0.3.0/cirros-0.3.0-i386-disk.img",
			},
			PVC: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: resource.MustParse("3Gi"),
					},
				},
				StorageClassName: &storageClassName,
			},
			SnapshotClassName: "csi-rbdplugin-snapclass",
		},
	}

	f := framework.Global
	err = f.Client.Create(context.Background(), vmi, cleanupOptions)
	if err != nil {
		t.Log(err)
		t.Fatal(err)
	}

	if err := waitForVmi(t, ns, "testcr", retryInterval, timeout); err != nil {
		t.Log(err)
		t.Fatal(err)
	}
}

func waitForVmi(t *testing.T, namespace, name string, retryInterval, timeout time.Duration) error {
	f := framework.Global
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		vmi := &v1alpha1.VirtualMachineImage{}
		err2 := f.Client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, vmi)
		if err2 != nil {
			if errors.IsNotFound(err2) {
				t.Logf("Waiting for creating vmi: %s in Namespace: %s \n", name, namespace)
				return false, nil
			}
			return false, err2
		}

		if vmi.Status.State == v1alpha1.VirtualMachineImageStateAvailable {
			return true, nil
		}
		t.Logf("Waiting for full availability of %s vmi\n", name)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Vmi available (%s/%s)\n", namespace, name)
	return nil
}

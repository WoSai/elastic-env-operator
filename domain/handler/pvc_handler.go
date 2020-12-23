package handler

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/gogo/protobuf/proto"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type pvcHandler struct {
	sqbdeployment *qav1alpha1.SQBDeployment
	ctx           context.Context
}

func NewPVCHandler(sqbdeployment *qav1alpha1.SQBDeployment, ctx context.Context) *pvcHandler {
	return &pvcHandler{sqbdeployment: sqbdeployment, ctx: ctx}
}

func (h *pvcHandler) CreateOrUpdate() error {
	exists := make(map[string]struct{}, 0)
	pvcList, err := h.getPVCList()
	if err != nil {
		return err
	}
	for _, pvc := range pvcList.Items {
		exists[pvc.Name] = struct{}{}
	}
	for _, volumespec := range h.sqbdeployment.Spec.Volumes {
		if !volumespec.PersistentVolumeClaim {
			continue
		}
		pvcName := volumespec.PersistentVolumeClaimName
		if pvcName == "" {
			pvcName = h.getPVCName(volumespec.MountPath)
			volumespec.PersistentVolumeClaimName = pvcName
		}
		if _, ok := exists[pvcName]; ok {
			delete(exists, pvcName)
			continue
		}
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: h.sqbdeployment.Namespace,
				Name:      pvcName,
			},
		}
		err = k8sclient.Get(h.ctx, client.ObjectKey{Namespace: pvc.Namespace, Name: pvc.Name}, pvc)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteMany,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("2Gi"),
					},
				},
				StorageClassName: proto.String("ack" + "-" + h.sqbdeployment.Labels[entity.GroupKey]),
			}
			pvc.Labels = h.sqbdeployment.Labels
			if err = CreateOrUpdate(h.ctx, pvc); err != nil {
				return err
			}
		}
	}
	for pvcName := range exists {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: h.sqbdeployment.Namespace,
				Name:      pvcName,
			},
		}
		if err = Delete(h.ctx, pvc); err != nil {
			return err
		}
	}
	return nil
}

func (h *pvcHandler) Delete() error {
	pvcList, err := h.getPVCList()
	if err != nil {
		return err
	}
	for _, pvc := range pvcList.Items {
		if Delete(h.ctx, &pvc) != nil {
			return err
		}
	}
	return nil
}

func (h *pvcHandler) getPVCList() (*corev1.PersistentVolumeClaimList, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	err := k8sclient.List(h.ctx, pvcList, &client.ListOptions{LabelSelector: labels.SelectorFromSet(map[string]string{
		entity.AppKey:   h.sqbdeployment.Spec.Selector.App,
		entity.PlaneKey: h.sqbdeployment.Spec.Selector.Plane,
	})})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	return pvcList, nil
}

func (h *pvcHandler) getPVCName(mountPath string) string {
	hash := md5.Sum([]byte(mountPath + h.sqbdeployment.CreationTimestamp.String()))
	return h.sqbdeployment.Spec.Selector.App + "-" + h.sqbdeployment.Spec.Selector.Plane + "-" + fmt.Sprintf("%x", hash)
}

func (h *pvcHandler) Handle() error {
	if deleted, _ := IsDeleted(h.sqbdeployment); deleted {
		return h.Delete()
	}
	return h.CreateOrUpdate()
}

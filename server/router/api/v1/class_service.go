package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

// ClassService handles class-related operations.
// Implements v1pb.ClassServiceServer (to be generated from protobuf).
// TODO: Add v1pb.UnimplementedClassServiceServer embedding once proto is defined.

// CreateClass creates a new class.
func (s *APIV1Service) CreateClass(ctx context.Context, request *v1pb.CreateClassRequest) (*v1pb.Class, error) {
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	// Validate request
	if request.Class == nil {
		return nil, status.Errorf(codes.InvalidArgument, "class is required")
	}
	if request.Class.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "class name is required")
	}

	// TODO: Generate class ID (shortuuid or custom)
	classUID := strings.TrimSpace(request.ClassId)
	if classUID == "" {
		// Generate unique ID
		classUID = "class_" + strings.ToLower(strings.ReplaceAll(request.Class.Name, " ", "_"))
		// TODO: Ensure uniqueness
	}

	// Convert protobuf Class to store Class
	class := &store.Class{
		UID:         classUID,
		Name:        request.Class.Name,
		Description: request.Class.Description,
		CreatorID:   user.ID,
		// TODO: Set other fields: invite_code, settings, visibility, etc.
	}

	// TODO: Check permissions (only teachers/admins can create classes)
	// TODO: Validate class settings

	createdClass, err := s.Store.CreateClass(ctx, class)
	if err != nil {
		// Check for duplicate
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return nil, status.Errorf(codes.AlreadyExists, "class with ID %q already exists", classUID)
		}
		return nil, status.Errorf(codes.Internal, "failed to create class: %v", err)
	}

	// Convert store Class to protobuf Class
	classMessage, err := s.convertClassFromStore(ctx, createdClass)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert class")
	}

	slog.Info("Class created", slog.String("uid", createdClass.UID), slog.String("name", createdClass.Name))
	return classMessage, nil
}

// GetClass retrieves a class by name.
func (s *APIV1Service) GetClass(ctx context.Context, request *v1pb.GetClassRequest) (*v1pb.Class, error) {
	classUID, err := ExtractClassUIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class name: %v", err)
	}

	class, err := s.Store.GetClass(ctx, &store.FindClass{
		UID: &classUID,
	})
	if err != nil {
		return nil, err
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}

	// TODO: Check visibility/permissions
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	if user == nil {
		// Public classes only for unauthenticated users
		if class.Visibility != store.ClassVisibilityPublic {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied")
		}
	} else {
		// TODO: Check if user is member or has access
	}

	classMessage, err := s.convertClassFromStore(ctx, class)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert class")
	}
	return classMessage, nil
}

// ListClasses lists classes based on filters.
func (s *APIV1Service) ListClasses(ctx context.Context, request *v1pb.ListClassesRequest) (*v1pb.ListClassesResponse, error) {
	classFind := &store.FindClass{}

	// Apply filters
	if request.Filter != "" {
		// TODO: Parse filter string
		classFind.Filters = append(classFind.Filters, request.Filter)
	}

	// TODO: Handle pagination
	var limit, offset int
	if request.PageToken != "" {
		// TODO: Parse page token
	} else {
		limit = int(request.PageSize)
	}
	if limit <= 0 {
		limit = DefaultPageSize
	}
	limitPlusOne := limit + 1
	classFind.Limit = &limitPlusOne
	classFind.Offset = &offset

	// TODO: Apply visibility/permission filters based on current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	if currentUser == nil {
		// Only public classes for unauthenticated users
		classFind.Visibility = store.ClassVisibilityPublic
	} else {
		// Show classes where user is member or public/protected
		// TODO: Implement member-based filtering
	}

	classes, err := s.Store.ListClasses(ctx, classFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list classes: %v", err)
	}

	classMessages := []*v1pb.Class{}
	nextPageToken := ""
	if len(classes) == limitPlusOne {
		classes = classes[:limit]
		nextPageToken, err = getPageToken(limit, offset+limit)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
	}

	for _, class := range classes {
		classMessage, err := s.convertClassFromStore(ctx, class)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert class")
		}
		classMessages = append(classMessages, classMessage)
	}

	response := &v1pb.ListClassesResponse{
		Classes:       classMessages,
		NextPageToken: nextPageToken,
	}
	return response, nil
}

// UpdateClass updates a class.
func (s *APIV1Service) UpdateClass(ctx context.Context, request *v1pb.UpdateClassRequest) (*v1pb.Class, error) {
	classUID, err := ExtractClassUIDFromName(request.Class.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class name: %v", err)
	}
	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update mask is required")
	}

	class, err := s.Store.GetClass(ctx, &store.FindClass{UID: &classUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}

	// TODO: Check permissions (only teachers/admins can update)
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if class.CreatorID != user.ID && !isSuperUser(user) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	update := &store.UpdateClass{
		ID: class.ID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "name":
			if request.Class.Name == "" {
				return nil, status.Errorf(codes.InvalidArgument, "class name cannot be empty")
			}
			update.Name = &request.Class.Name
		case "description":
			update.Description = &request.Class.Description
		case "settings":
			// TODO: Convert protobuf settings to store settings
		case "visibility":
			// TODO: Convert protobuf visibility to store visibility
		case "invite_code":
			// TODO: Handle invite code regeneration
		}
	}

	if err = s.Store.UpdateClass(ctx, update); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update class")
	}

	updatedClass, err := s.Store.GetClass(ctx, &store.FindClass{
		ID: &class.ID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get class")
	}

	classMessage, err := s.convertClassFromStore(ctx, updatedClass)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert class")
	}
	return classMessage, nil
}

// DeleteClass deletes a class.
func (s *APIV1Service) DeleteClass(ctx context.Context, request *v1pb.DeleteClassRequest) (*emptypb.Empty, error) {
	classUID, err := ExtractClassUIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class name: %v", err)
	}

	class, err := s.Store.GetClass(ctx, &store.FindClass{
		UID: &classUID,
	})
	if err != nil {
		return nil, err
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}

	// TODO: Check permissions
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if class.CreatorID != user.ID && !isSuperUser(user) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	// TODO: Check if class has members? Maybe require force flag

	if err = s.Store.DeleteClass(ctx, &store.DeleteClass{ID: class.ID}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete class")
	}

	slog.Info("Class deleted", slog.String("uid", class.UID), slog.String("name", class.Name))
	return &emptypb.Empty{}, nil
}

// convertClassFromStore converts a store.Class to a v1pb.Class.
func (s *APIV1Service) convertClassFromStore(ctx context.Context, class *store.Class) (*v1pb.Class, error) {
	if class == nil {
		return nil, errors.New("class is nil")
	}

	// TODO: Fetch creator information
	creatorName := fmt.Sprintf("%s%d", UserNamePrefix, class.CreatorID)

	// TODO: Convert settings, visibility, etc.
	return &v1pb.Class{
		Name:        fmt.Sprintf("%s%s", ClassNamePrefix, class.UID),
		DisplayName: class.Name,
		Description: class.Description,
		Creator:     creatorName,
		// TODO: Fill other fields
	}, nil
}

// ExtractClassUIDFromName extracts class UID from resource name.
func ExtractClassUIDFromName(name string) (string, error) {
	if !strings.HasPrefix(name, ClassNamePrefix) {
		return "", errors.Errorf("invalid class name prefix")
	}
	return strings.TrimPrefix(name, ClassNamePrefix), nil
}

// Constants for class resource names.
const (
	ClassNamePrefix = "classes/"
)

// TODO: Add methods for class members, memo visibility, tag templates, etc.
// - AddClassMember
// - RemoveClassMember
// - ListClassMembers
// - UpdateClassMemberRole
// - SetClassMemoVisibility
// - GetClassMemoVisibility
// - ListClassMemoVisibility
// - CreateClassTagTemplate
// - UpdateClassTagTemplate
// - DeleteClassTagTemplate
// - ListClassTagTemplates
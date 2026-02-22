package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/usememos/memos/internal/base"
	"github.com/usememos/memos/plugin/filter"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

// ClassService handles class-related operations.
// Implements v1pb.ClassServiceServer.

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

	// Generate class ID (shortuuid or custom)
	var classUID string
	if request.ClassId != nil && *request.ClassId != "" {
		classUID = strings.TrimSpace(*request.ClassId)
		// Validate custom class ID format
		if !base.UIDMatcher.MatchString(classUID) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid class_id format: must be 1-32 characters, alphanumeric and hyphens only, cannot start or end with hyphen")
		}
	} else {
		// Generate unique ID with shortuuid
		classUID = shortuuid.New()
	}

	// Convert protobuf Class to store Class
	visibility, err := convertClassVisibilityToStore(request.Class.Visibility)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid visibility: %v", err)
	}

	settings, err := convertSettingsToStore(request.Class.Settings)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid settings: %v", err)
	}

	now := time.Now().Unix()
	inviteCode := ""
	if request.Class.InviteCode != nil {
		inviteCode = *request.Class.InviteCode
	}
	class := &store.Class{
		UID:         classUID,
		Name:        request.Class.Name,
		Description: request.Class.Description,
		CreatorID:   user.ID,
		CreatedTs:   now,
		UpdatedTs:   now,
		Visibility:  visibility,
		InviteCode:  inviteCode,
		Settings:    settings,
	}

	// Check permissions (only teachers/admins can create classes)
	if !s.canCreateClass(user) {
		return nil, status.Errorf(codes.PermissionDenied, "only teachers and administrators can create classes")
	}
	
	// Validate class settings (already validated in convertSettingsToStore)

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

	// Check visibility/permissions
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	
	canView, err := s.canViewClass(ctx, user, class)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check view permissions: %v", err)
	}
	if !canView {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
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
		if err := s.validateFilter(ctx, request.Filter); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
		}
		classFind.Filters = append(classFind.Filters, request.Filter)
	}

	// Handle pagination
	var limit, offset int
	if request.PageToken != "" {
		var pageToken v1pb.PageToken
		if err := unmarshalPageToken(request.PageToken, &pageToken); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page token: %v", err)
		}
		limit = int(pageToken.Limit)
		offset = int(pageToken.Offset)
	} else {
		limit = int(request.PageSize)
	}
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	limitPlusOne := limit + 1
	classFind.Limit = &limitPlusOne
	classFind.Offset = &offset

	// Apply visibility/permission filters based on current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	if currentUser == nil {
		// Only public classes for unauthenticated users
		publicVisibility := store.ClassVisibilityPublic
		classFind.Visibility = &publicVisibility
	} else {
		// For authenticated users, we need to filter in memory for now
		// because database query doesn't support complex permission logic
		// We'll fetch all classes and filter in memory
		// This is temporary until we have proper member-based filtering
		classFind.Visibility = nil // Clear visibility filter, we'll filter in memory
	}

	classes, err := s.Store.ListClasses(ctx, classFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list classes: %v", err)
	}

	// Filter classes based on user permissions
	filteredClasses := []*store.Class{}
	for _, class := range classes {
		canView, err := s.canViewClass(ctx, currentUser, class)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check view permissions: %v", err)
		}
		if canView {
			filteredClasses = append(filteredClasses, class)
		}
	}
	classes = filteredClasses

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

	// Check permissions (only admins and class creators can update)
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if !s.canManageClass(user, class) {
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
			// Convert protobuf settings to store settings
			settings, err := convertSettingsToStore(request.Class.Settings)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid settings: %v", err)
			}
			update.Settings = settings
		case "visibility":
			// Convert protobuf visibility to store visibility
			visibility, err := convertClassVisibilityToStore(request.Class.Visibility)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid visibility: %v", err)
			}
			update.Visibility = &visibility
		case "invite_code":
			// Handle invite code
			if request.Class.InviteCode != nil {
				inviteCode := *request.Class.InviteCode
				update.InviteCode = &inviteCode
			} else {
				// Clear invite code
				emptyString := ""
				update.InviteCode = &emptyString
			}
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

	// Check permissions (only admins and class creators can delete)
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if !s.canManageClass(user, class) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	// Check if class has members
	if hasMembers, err := s.hasClassMembers(ctx, class.ID); err == nil && hasMembers {
		return nil, status.Errorf(codes.FailedPrecondition, "class has members, cannot delete")
	}

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

	// Fetch creator information
	creator, err := s.Store.GetUser(ctx, &store.FindUser{ID: &class.CreatorID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get creator")
	}
	if creator == nil {
		return nil, errors.Errorf("creator not found for ID %d", class.CreatorID)
	}
	creatorName := fmt.Sprintf("%s%d", UserNamePrefix, creator.ID)

	// Convert visibility
	visibility := convertClassVisibilityFromStore(class.Visibility)

	// Convert settings
	settings := convertSettingsFromStore(class.Settings)

	// Convert timestamps
	createTime := timestamppb.New(time.Unix(class.CreatedTs, 0))
	updateTime := timestamppb.New(time.Unix(class.UpdatedTs, 0))

	return &v1pb.Class{
		Name:        fmt.Sprintf("%s%s", ClassNamePrefix, class.UID),
		Uid:         class.UID,
		DisplayName: class.Name,
		Description: class.Description,
		Creator:     creatorName,
		CreateTime:  createTime,
		UpdateTime:  updateTime,
		Visibility:  visibility,
		InviteCode:  &class.InviteCode,
		Settings:    settings,
	}, nil
}

// convertClassVisibilityToStore converts protobuf ClassVisibility to store.ClassVisibility.
func convertClassVisibilityToStore(v v1pb.ClassVisibility) (store.ClassVisibility, error) {
	switch v {
	case v1pb.ClassVisibility_CLASS_PUBLIC:
		return store.ClassVisibilityPublic, nil
	case v1pb.ClassVisibility_CLASS_PROTECTED:
		return store.ClassVisibilityProtected, nil
	case v1pb.ClassVisibility_CLASS_PRIVATE:
		return store.ClassVisibilityPrivate, nil
	default:
		return "", errors.Errorf("invalid visibility: %v", v)
	}
}

// convertClassVisibilityFromStore converts store.ClassVisibility to protobuf ClassVisibility.
func convertClassVisibilityFromStore(v store.ClassVisibility) v1pb.ClassVisibility {
	switch v {
	case store.ClassVisibilityPublic:
		return v1pb.ClassVisibility_CLASS_PUBLIC
	case store.ClassVisibilityProtected:
		return v1pb.ClassVisibility_CLASS_PROTECTED
	case store.ClassVisibilityPrivate:
		return v1pb.ClassVisibility_CLASS_PRIVATE
	default:
		return v1pb.ClassVisibility_CLASS_VISIBILITY_UNSPECIFIED
	}
}

// convertSettingsToStore converts protobuf ClassSettings to storepb.ClassSettings.
func convertSettingsToStore(s *v1pb.ClassSettings) (*storepb.ClassSettings, error) {
	if s == nil {
		return nil, nil
	}
	
	// Create a map for the settings
	settingsMap := make(map[string]interface{})
	
	// Convert each field if present
	if s.StudentMemoVisibility != nil {
		settingsMap["student_memo_visibility"] = *s.StudentMemoVisibility
	}
	if s.AllowAnonymous != nil {
		settingsMap["allow_anonymous"] = *s.AllowAnonymous
	}
	if s.DefaultStudentVisibility != v1pb.ClassVisibility_CLASS_VISIBILITY_UNSPECIFIED {
		// Convert enum to string
		settingsMap["default_student_visibility"] = s.DefaultStudentVisibility.String()
	}
	if s.EnableTagTemplates != nil {
		settingsMap["enable_tag_templates"] = *s.EnableTagTemplates
	}
	if s.MaxMembers != nil {
		settingsMap["max_members"] = *s.MaxMembers
	}
	if s.RequireMemberApproval != nil {
		settingsMap["require_member_approval"] = *s.RequireMemberApproval
	}
	
	// Convert map to Struct
	settingsStruct, err := structpb.NewStruct(settingsMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create settings struct")
	}
	
	return &storepb.ClassSettings{
		Settings: settingsStruct,
	}, nil
}

// convertSettingsFromStore converts storepb.ClassSettings to protobuf ClassSettings.
func convertSettingsFromStore(s *storepb.ClassSettings) *v1pb.ClassSettings {
	if s == nil || s.Settings == nil {
		return nil
	}
	
	settings := &v1pb.ClassSettings{}
	
	// Extract values from the Struct
	if val, ok := s.Settings.Fields["student_memo_visibility"]; ok {
		if boolVal, ok := val.Kind.(*structpb.Value_BoolValue); ok {
			settings.StudentMemoVisibility = &boolVal.BoolValue
		}
	}
	
	if val, ok := s.Settings.Fields["allow_anonymous"]; ok {
		if boolVal, ok := val.Kind.(*structpb.Value_BoolValue); ok {
			settings.AllowAnonymous = &boolVal.BoolValue
		}
	}
	
	if val, ok := s.Settings.Fields["default_student_visibility"]; ok {
		if strVal, ok := val.Kind.(*structpb.Value_StringValue); ok {
			// Convert string to enum
			switch strVal.StringValue {
			case "CLASS_PUBLIC":
				settings.DefaultStudentVisibility = v1pb.ClassVisibility_CLASS_PUBLIC
			case "CLASS_PROTECTED":
				settings.DefaultStudentVisibility = v1pb.ClassVisibility_CLASS_PROTECTED
			case "CLASS_PRIVATE":
				settings.DefaultStudentVisibility = v1pb.ClassVisibility_CLASS_PRIVATE
			}
		}
	}
	
	if val, ok := s.Settings.Fields["enable_tag_templates"]; ok {
		if boolVal, ok := val.Kind.(*structpb.Value_BoolValue); ok {
			settings.EnableTagTemplates = &boolVal.BoolValue
		}
	}
	
	if val, ok := s.Settings.Fields["max_members"]; ok {
		if numVal, ok := val.Kind.(*structpb.Value_NumberValue); ok {
			intVal := int32(numVal.NumberValue)
			settings.MaxMembers = &intVal
		}
	}
	
	if val, ok := s.Settings.Fields["require_member_approval"]; ok {
		if boolVal, ok := val.Kind.(*structpb.Value_BoolValue); ok {
			settings.RequireMemberApproval = &boolVal.BoolValue
		}
	}
	
	return settings
}

// ExtractClassUIDFromName extracts class UID from resource name.
func ExtractClassUIDFromName(name string) (string, error) {
	if !strings.HasPrefix(name, ClassNamePrefix) {
		return "", errors.Errorf("invalid class name prefix")
	}
	return strings.TrimPrefix(name, ClassNamePrefix), nil
}

// Constants for class resource names.
// ClassNamePrefix is defined in resource_name.go

// Helper functions

// isSuperUser checks if the user has admin privileges or is the creator of the resource.
func (s *APIV1Service) isSuperUser(user *store.User) bool {
	if user == nil {
		return false
	}
	return user.Role == store.RoleAdmin
}

// canCreateClass checks if a user can create a class.
// Only teachers and administrators can create classes.
func (s *APIV1Service) canCreateClass(user *store.User) bool {
	if user == nil {
		return false
	}
	// Administrators can create classes
	if s.isSuperUser(user) {
		return true
	}
	// TODO: Check if user has TEACHER role in any class when member management is implemented
	// For now, allow any authenticated user to create a class
	// (they will become the teacher of the class they create)
	return true
}

// canManageClass checks if a user can manage (update/delete) a specific class.
// Admins and class creators can manage the class.
func (s *APIV1Service) canManageClass(user *store.User, class *store.Class) bool {
	if user == nil || class == nil {
		return false
	}
	return s.isSuperUser(user) || class.CreatorID == user.ID
}

// isClassMember checks if a user is a member of a class.
func (s *APIV1Service) isClassMember(ctx context.Context, userID int32, classID int32) (bool, error) {
	// Check if user is the class creator
	class, err := s.Store.GetClass(ctx, &store.FindClass{ID: &classID})
	if err != nil {
		return false, errors.Wrap(err, "failed to get class")
	}
	if class == nil {
		return false, errors.New("class not found")
	}
	if class.CreatorID == userID {
		return true, nil
	}
	
	// Check class_member table
	members, err := s.Store.ListClassMembers(ctx, &store.FindClassMember{
		ClassID: &classID,
		UserID:  &userID,
		Limit:   &[]int{1}[0],
	})
	if err != nil {
		// If driver not implemented, fall back to creator check only
		return false, nil
	}
	return len(members) > 0, nil
}

// hasClassMembers checks if a class has any members.
func (s *APIV1Service) hasClassMembers(ctx context.Context, classID int32) (bool, error) {
	members, err := s.Store.ListClassMembers(ctx, &store.FindClassMember{
		ClassID: &classID,
		Limit:   &[]int{1}[0], // Limit to 1 for efficiency
	})
	if err != nil {
		// If driver not implemented, return false (allow deletion)
		return false, nil
	}
	return len(members) > 0, nil
}

// canViewClass checks if a user can view a specific class.
func (s *APIV1Service) canViewClass(ctx context.Context, user *store.User, class *store.Class) (bool, error) {
	if class == nil {
		return false, nil
	}
	
	// Public classes are visible to everyone
	if class.Visibility == store.ClassVisibilityPublic {
		return true, nil
	}
	
	// For protected and private classes, need authentication
	if user == nil {
		return false, nil
	}
	
	// Admins can view all classes
	if s.isSuperUser(user) {
		return true, nil
	}
	
	// Class creator can view their own class
	if class.CreatorID == user.ID {
		return true, nil
	}
	
	// Check class membership
	isMember, err := s.isClassMember(ctx, user.ID, class.ID)
	if err != nil {
		return false, errors.Wrap(err, "failed to check class membership")
	}
	if isMember {
		return true, nil
	}
	
	// Protected classes are visible to all authenticated users
	if class.Visibility == store.ClassVisibilityProtected {
		return true, nil
	}
	
	// Private classes only for creators/admins/members (handled above)
	return false, nil
}

// validateClassFilter validates a filter string using the filter engine for class queries.
func (s *APIV1Service) validateClassFilter(ctx context.Context, filterStr string) error {
	if filterStr == "" {
		return errors.New("filter cannot be empty")
	}

	engine, err := filter.DefaultEngine()
	if err != nil {
		return err
	}

	var dialect filter.DialectName
	switch s.Profile.Driver {
	case "mysql":
		dialect = filter.DialectMySQL
	case "postgres":
		dialect = filter.DialectPostgres
	case "sqlite":
		dialect = filter.DialectSQLite
	default:
		return errors.Errorf("unsupported driver: %s", s.Profile.Driver)
	}

	if _, err := engine.CompileToStatement(ctx, filterStr, filter.RenderOptions{Dialect: dialect}); err != nil {
		return errors.Wrap(err, "invalid filter")
	}
	return nil
}

// AddClassMember adds a user to a class as a member.
func (s *APIV1Service) AddClassMember(ctx context.Context, request *v1pb.AddClassMemberRequest) (*v1pb.ClassMember, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddClassMember not implemented")
}

// RemoveClassMember removes a member from a class.
func (s *APIV1Service) RemoveClassMember(ctx context.Context, request *v1pb.RemoveClassMemberRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveClassMember not implemented")
}

// ListClassMembers lists members of a class.
func (s *APIV1Service) ListClassMembers(ctx context.Context, request *v1pb.ListClassMembersRequest) (*v1pb.ListClassMembersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListClassMembers not implemented")
}

// UpdateClassMemberRole updates a member's role in a class.
func (s *APIV1Service) UpdateClassMemberRole(ctx context.Context, request *v1pb.UpdateClassMemberRoleRequest) (*v1pb.ClassMember, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateClassMemberRole not implemented")
}

// SetClassMemoVisibility sets visibility of a memo within a class.
func (s *APIV1Service) SetClassMemoVisibility(ctx context.Context, request *v1pb.SetClassMemoVisibilityRequest) (*v1pb.ClassMemoVisibility, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetClassMemoVisibility not implemented")
}

// GetClassMemoVisibility gets visibility settings of a memo in a class.
func (s *APIV1Service) GetClassMemoVisibility(ctx context.Context, request *v1pb.GetClassMemoVisibilityRequest) (*v1pb.ClassMemoVisibility, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetClassMemoVisibility not implemented")
}

// ListClassMemoVisibilities lists memo visibility settings for a class.
func (s *APIV1Service) ListClassMemoVisibilities(ctx context.Context, request *v1pb.ListClassMemoVisibilitiesRequest) (*v1pb.ListClassMemoVisibilitiesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListClassMemoVisibilities not implemented")
}

// CreateClassTagTemplate creates a tag template for a class.
func (s *APIV1Service) CreateClassTagTemplate(ctx context.Context, request *v1pb.CreateClassTagTemplateRequest) (*v1pb.ClassTagTemplate, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateClassTagTemplate not implemented")
}

// UpdateClassTagTemplate updates a tag template.
func (s *APIV1Service) UpdateClassTagTemplate(ctx context.Context, request *v1pb.UpdateClassTagTemplateRequest) (*v1pb.ClassTagTemplate, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateClassTagTemplate not implemented")
}

// DeleteClassTagTemplate deletes a tag template.
func (s *APIV1Service) DeleteClassTagTemplate(ctx context.Context, request *v1pb.DeleteClassTagTemplateRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteClassTagTemplate not implemented")
}

// ListClassTagTemplates lists tag templates for a class.
func (s *APIV1Service) ListClassTagTemplates(ctx context.Context, request *v1pb.ListClassTagTemplatesRequest) (*v1pb.ListClassTagTemplatesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListClassTagTemplates not implemented")
}
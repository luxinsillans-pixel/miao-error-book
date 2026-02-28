package v1

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
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
	"github.com/usememos/memos/internal/util"
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
	
	// Generate a random invite code if not provided
	if inviteCode == "" {
		// Generate 8-character alphanumeric invite code
		inviteCode = generateInviteCode(8)
	}
	
	// Determine display name: use DisplayName if provided, otherwise fall back to Name
	displayName := request.Class.DisplayName
	if displayName == "" {
		// If DisplayName is empty, use Name as fallback
		displayName = request.Class.Name
	}
	
	class := &store.Class{
		UID:         classUID,
		Name:        displayName,  // Store display name in the Name field
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
			// For backward compatibility, treat "name" as display_name
			// since store.Class.Name stores the display name
			displayName := request.Class.DisplayName
			if displayName == "" {
				// Fallback to Name field if DisplayName is not set
				displayName = request.Class.Name
			}
			if displayName == "" {
				return nil, status.Errorf(codes.InvalidArgument, "class name cannot be empty")
			}
			update.Name = &displayName
		case "display_name":
			if request.Class.DisplayName == "" {
				return nil, status.Errorf(codes.InvalidArgument, "class display_name cannot be empty")
			}
			update.Name = &request.Class.DisplayName
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
	fmt.Fprintf(os.Stderr, "DEBUG convertClassVisibilityToStore called with v=%v (int=%d, string=%s)\n", v, int32(v), v.String())
	switch v {
	case v1pb.ClassVisibility_CLASS_PUBLIC:
		fmt.Fprintf(os.Stderr, "DEBUG Case CLASS_PUBLIC, returning store.ClassVisibilityPublic=%q\n", store.ClassVisibilityPublic)
		return store.ClassVisibilityPublic, nil
	case v1pb.ClassVisibility_CLASS_PROTECTED:
		fmt.Fprintf(os.Stderr, "DEBUG Case CLASS_PROTECTED, returning store.ClassVisibilityProtected=%q\n", store.ClassVisibilityProtected)
		return store.ClassVisibilityProtected, nil
	case v1pb.ClassVisibility_CLASS_PRIVATE:
		fmt.Fprintf(os.Stderr, "DEBUG Case CLASS_PRIVATE, returning store.ClassVisibilityPrivate=%q\n", store.ClassVisibilityPrivate)
		return store.ClassVisibilityPrivate, nil
	default:
		fmt.Fprintf(os.Stderr, "DEBUG Default case, invalid visibility: %v\n", v)
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
	// Extract class UID from class resource name
	classUID, err := ExtractClassUIDFromName(request.Class)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class name: %v", err)
	}
	
	// Extract user ID from user resource name
	userID, err := ExtractUserIDFromName(request.User)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user name: %v", err)
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{UID: &classUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check permissions: only class teachers/admins can add members
	if !s.canManageClass(currentUser, class) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied: only class teachers and administrators can add members")
	}
	
	// Check if user is already a member (including as creator)
	isMember, err := s.isClassMember(ctx, userID, class.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check membership: %v", err)
	}
	if isMember {
		return nil, status.Errorf(codes.AlreadyExists, "user is already a member of this class")
	}
	
	// Convert role
	role, err := convertClassMemberRoleToStore(request.Role)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid role: %v", err)
	}
	
	// Create class member
	now := time.Now().Unix()
	classMember := &store.ClassMember{
		ClassID:   class.ID,
		UserID:    userID,
		Role:      role,
		JoinedTs:  now,
		InvitedBy: &currentUser.ID,
	}
	
	createdMember, err := s.Store.CreateClassMember(ctx, classMember)
	if err != nil {
		// Check for duplicate
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return nil, status.Errorf(codes.AlreadyExists, "user is already a member of this class")
		}
		return nil, status.Errorf(codes.Internal, "failed to add class member: %v", err)
	}
	
	// Convert to protobuf response
	memberMessage, err := s.convertClassMemberFromStore(ctx, createdMember)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert class member")
	}
	
	slog.Info("Class member added", 
		slog.String("class", class.UID), 
		slog.Int("user_id", int(userID)),
		slog.String("role", string(role)))
	
	return memberMessage, nil
}

// RemoveClassMember removes a member from a class.
func (s *APIV1Service) RemoveClassMember(ctx context.Context, request *v1pb.RemoveClassMemberRequest) (*emptypb.Empty, error) {
	// Extract class member ID from resource name
	memberID, err := ExtractClassMemberIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class member name: %v", err)
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get class member
	classMember, err := s.Store.GetClassMember(ctx, &store.FindClassMember{ID: &memberID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class member: %v", err)
	}
	if classMember == nil {
		return nil, status.Errorf(codes.NotFound, "class member not found")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{ID: &classMember.ClassID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check permissions: only class teachers/admins can remove members
	if !s.canManageClass(currentUser, class) {
		// Also allow users to remove themselves
		if currentUser.ID != classMember.UserID {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied: only class teachers, administrators, or the member themselves can remove members")
		}
	}
	
	// Check if trying to remove class creator (shouldn't happen through class_member table)
	if class.CreatorID == classMember.UserID {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot remove class creator from class")
	}
	
	// Delete class member
	if err = s.Store.DeleteClassMember(ctx, &store.DeleteClassMember{ID: classMember.ID}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove class member: %v", err)
	}
	
	slog.Info("Class member removed", 
		slog.String("class", class.UID), 
		slog.Int("user_id", int(classMember.UserID)),
		slog.Int("member_id", int(memberID)))
	
	return &emptypb.Empty{}, nil
}

// ListClassMembers lists members of a class.
func (s *APIV1Service) ListClassMembers(ctx context.Context, request *v1pb.ListClassMembersRequest) (*v1pb.ListClassMembersResponse, error) {
	// Extract class UID from class resource name
	classUID, err := ExtractClassUIDFromName(request.Class)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class name: %v", err)
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{UID: &classUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check if user can view the class
	canView, err := s.canViewClass(ctx, currentUser, class)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check view permissions: %v", err)
	}
	if !canView {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied: cannot view this class")
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
	
	// Find class members
	classMemberFind := &store.FindClassMember{
		ClassID: &class.ID,
		Limit:   &limitPlusOne,
		Offset:  &offset,
	}
	
	members, err := s.Store.ListClassMembers(ctx, classMemberFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list class members: %v", err)
	}
	
	// Convert to protobuf messages
	memberMessages := []*v1pb.ClassMember{}
	nextPageToken := ""
	if len(members) == limitPlusOne {
		members = members[:limit]
		nextPageToken, err = getPageToken(limit, offset+limit)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
	}
	
	for _, member := range members {
		memberMessage, err := s.convertClassMemberFromStore(ctx, member)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert class member")
		}
		memberMessages = append(memberMessages, memberMessage)
	}
	
	response := &v1pb.ListClassMembersResponse{
		Members:       memberMessages,
		NextPageToken: nextPageToken,
	}
	return response, nil
}

// UpdateClassMemberRole updates a member's role in a class.
func (s *APIV1Service) UpdateClassMemberRole(ctx context.Context, request *v1pb.UpdateClassMemberRoleRequest) (*v1pb.ClassMember, error) {
	// Extract class member ID from resource name
	memberID, err := ExtractClassMemberIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class member name: %v", err)
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get class member
	classMember, err := s.Store.GetClassMember(ctx, &store.FindClassMember{ID: &memberID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class member: %v", err)
	}
	if classMember == nil {
		return nil, status.Errorf(codes.NotFound, "class member not found")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{ID: &classMember.ClassID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check permissions: only class teachers/admins can update member roles
	if !s.canManageClass(currentUser, class) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied: only class teachers and administrators can update member roles")
	}
	
	// Check if trying to update class creator (shouldn't happen through class_member table)
	if class.CreatorID == classMember.UserID {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot change role of class creator")
	}
	
	// Convert role
	newRole, err := convertClassMemberRoleToStore(request.Role)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid role: %v", err)
	}
	
	// Update class member
	update := &store.UpdateClassMember{
		ID:   classMember.ID,
		Role: &newRole,
	}
	
	if err = s.Store.UpdateClassMember(ctx, update); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update class member role: %v", err)
	}
	
	// Get updated class member
	updatedMember, err := s.Store.GetClassMember(ctx, &store.FindClassMember{ID: &memberID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated class member: %v", err)
	}
	if updatedMember == nil {
		return nil, status.Errorf(codes.NotFound, "updated class member not found")
	}
	
	// Convert to protobuf response
	memberMessage, err := s.convertClassMemberFromStore(ctx, updatedMember)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert class member")
	}
	
	slog.Info("Class member role updated", 
		slog.String("class", class.UID), 
		slog.Int("user_id", int(classMember.UserID)),
		slog.String("old_role", string(classMember.Role)),
		slog.String("new_role", string(newRole)))
	
	return memberMessage, nil
}

// SetClassMemoVisibility sets visibility of a memo within a class.
func (s *APIV1Service) SetClassMemoVisibility(ctx context.Context, request *v1pb.SetClassMemoVisibilityRequest) (*v1pb.ClassMemoVisibility, error) {
	fmt.Fprintf(os.Stderr, "🚨🚨🚨 DEBUG SetClassMemoVisibility ENTER: class=%s, memo=%s, visibility=%v (%s, int=%d)\n", request.Class, request.Memo, request.Visibility, request.Visibility.String(), int32(request.Visibility))
	fmt.Fprintf(os.Stderr, "🚨🚨🚨 DEBUG Request: %+v\n", request)
	// Extract class UID from class resource name
	classUID, err := ExtractClassUIDFromName(request.Class)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class name: %v", err)
	}
	
	// Extract memo UID from memo resource name
	memoUID, err := ExtractMemoUIDFromName(request.Memo)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid memo name: %v", err)
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{UID: &classUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Get memo
	memo, err := s.Store.GetMemo(ctx, &store.FindMemo{UID: &memoUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get memo: %v", err)
	}
	if memo == nil {
		return nil, status.Errorf(codes.NotFound, "memo not found")
	}
	
	// Check permissions: user must be able to view the class and manage memos
	// For now, only class teachers/admins can set memo visibility
	if !s.canManageClass(currentUser, class) {
		// Also check if user is the memo creator and has permission to share
		if memo.CreatorID != currentUser.ID {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied: only class teachers, administrators, or memo creators can set memo visibility")
		}
		// Check if user is a class member (including as creator)
		isMember, err := s.isClassMember(ctx, currentUser.ID, class.ID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check membership: %v", err)
		}
		if !isMember {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied: must be a class member to share memos")
		}
	}
	
	// Convert visibility
	visibility, err := convertClassVisibilityToStore(request.Visibility)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid visibility: %v", err)
	}
	slog.Debug("Visibility conversion", 
		slog.String("request", request.Visibility.String()),
		slog.String("converted", string(visibility)))
	// Extra debug logging
	fmt.Fprintf(os.Stderr, "DEBUG SetClassMemoVisibility: request.Visibility=%v (%s), converted=%q (type: %T)\n", 
		request.Visibility, request.Visibility.String(), visibility, visibility)
	// Even more debug - print enum numeric value
	fmt.Fprintf(os.Stderr, "DEBUG Enum numeric value: %d\n", int32(request.Visibility))
	// Additional validation
	if visibility == "" {
		return nil, status.Errorf(codes.Internal, "converted visibility is empty")
	}
	// Check if it's a valid store.ClassVisibility value
	validValues := map[store.ClassVisibility]bool{
		store.ClassVisibilityPublic:    true,
		store.ClassVisibilityProtected: true,
		store.ClassVisibilityPrivate:   true,
	}
	if !validValues[visibility] {
		return nil, status.Errorf(codes.Internal, "invalid converted visibility value: %q", visibility)
	}
	// Debug: Print the actual bytes of the visibility string
	fmt.Fprintf(os.Stderr, "DEBUG visibility string bytes: %v\n", []byte(string(visibility)))
	fmt.Fprintf(os.Stderr, "DEBUG visibility string length: %d\n", len(string(visibility)))
	
	// Check if visibility record already exists
	existingVisibility, err := s.Store.GetClassMemoVisibility(ctx, &store.FindClassMemoVisibility{
		ClassID: &class.ID,
		MemoID:  &memo.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check existing visibility: %v", err)
	}
	
	now := time.Now().Unix()
	var createdVisibility *store.ClassMemoVisibility
	
	if existingVisibility != nil {
		// Update existing visibility
		update := &store.UpdateClassMemoVisibility{
			ID:         existingVisibility.ID,
			Visibility: &visibility,
		}
		
		if err = s.Store.UpdateClassMemoVisibility(ctx, update); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update memo visibility: %v", err)
		}
		
		// Get updated visibility
		createdVisibility, err = s.Store.GetClassMemoVisibility(ctx, &store.FindClassMemoVisibility{ID: &existingVisibility.ID})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get updated visibility: %v", err)
		}
	} else {
		// Create new visibility record
		visibilityRecord := &store.ClassMemoVisibility{
			ClassID:     class.ID,
			MemoID:      memo.ID,
			Visibility:  visibility,
			SharedBy:    currentUser.ID,
			SharedTs:    now,
			Description: "", // Could be extended to accept description in request
		}
		
		// DEBUG: Log the visibility value before creating
		fmt.Printf("🚨🚨🚨 DEBUG Before CreateClassMemoVisibility: visibility=%q (type: %T)\n", visibilityRecord.Visibility, visibilityRecord.Visibility)
		fmt.Fprintf(os.Stderr, "🚨🚨🚨 DEBUG Before CreateClassMemoVisibility: visibility=%q (type: %T)\n", visibilityRecord.Visibility, visibilityRecord.Visibility)
		slog.Debug("Before CreateClassMemoVisibility", slog.String("visibility", string(visibilityRecord.Visibility)))
		
		createdVisibility, err = s.Store.CreateClassMemoVisibility(ctx, visibilityRecord)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
				return nil, status.Errorf(codes.AlreadyExists, "memo visibility already set for this class")
			}
			return nil, status.Errorf(codes.Internal, "failed to set memo visibility: %v", err)
		}
	}
	
	if createdVisibility == nil {
		return nil, status.Errorf(codes.Internal, "failed to create or update memo visibility")
	}
	
	// Convert to protobuf response
	visibilityMessage, err := s.convertClassMemoVisibilityFromStore(ctx, createdVisibility)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert class memo visibility")
	}
	
	slog.Info("Class memo visibility set", 
		slog.String("class", class.UID), 
		slog.String("memo", memo.UID),
		slog.String("visibility", string(visibility)))
	
	return visibilityMessage, nil
}

// GetClassMemoVisibility gets visibility settings of a memo in a class.
func (s *APIV1Service) GetClassMemoVisibility(ctx context.Context, request *v1pb.GetClassMemoVisibilityRequest) (*v1pb.ClassMemoVisibility, error) {
	// Extract visibility ID from resource name
	visibilityID, err := ExtractClassMemoVisibilityIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class memo visibility name: %v", err)
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get visibility record
	visibility, err := s.Store.GetClassMemoVisibility(ctx, &store.FindClassMemoVisibility{ID: &visibilityID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class memo visibility: %v", err)
	}
	if visibility == nil {
		return nil, status.Errorf(codes.NotFound, "class memo visibility not found")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{ID: &visibility.ClassID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check if user can view the class
	canView, err := s.canViewClass(ctx, currentUser, class)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check view permissions: %v", err)
	}
	if !canView {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied: cannot view this class")
	}
	
	// Get memo to ensure it still exists (optional but good for consistency)
	memo, err := s.Store.GetMemo(ctx, &store.FindMemo{ID: &visibility.MemoID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get memo: %v", err)
	}
	if memo == nil {
		return nil, status.Errorf(codes.NotFound, "memo not found")
	}
	
	// Convert to protobuf response
	visibilityMessage, err := s.convertClassMemoVisibilityFromStore(ctx, visibility)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert class memo visibility")
	}
	
	return visibilityMessage, nil
}

// ListClassMemoVisibilities lists memo visibility settings for a class.
func (s *APIV1Service) ListClassMemoVisibilities(ctx context.Context, request *v1pb.ListClassMemoVisibilitiesRequest) (*v1pb.ListClassMemoVisibilitiesResponse, error) {
	// Extract class UID from class resource name
	classUID, err := ExtractClassUIDFromName(request.Class)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class name: %v", err)
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{UID: &classUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check if user can view the class
	canView, err := s.canViewClass(ctx, currentUser, class)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check view permissions: %v", err)
	}
	if !canView {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied: cannot view this class")
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
	
	// Find memo visibilities
	visibilityFind := &store.FindClassMemoVisibility{
		ClassID: &class.ID,
		Limit:   &limitPlusOne,
		Offset:  &offset,
	}
	
	visibilities, err := s.Store.ListClassMemoVisibilities(ctx, visibilityFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list class memo visibilities: %v", err)
	}
	
	// Convert to protobuf messages
	visibilityMessages := []*v1pb.ClassMemoVisibility{}
	nextPageToken := ""
	if len(visibilities) == limitPlusOne {
		visibilities = visibilities[:limit]
		nextPageToken, err = getPageToken(limit, offset+limit)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
	}
	
	for _, visibility := range visibilities {
		visibilityMessage, err := s.convertClassMemoVisibilityFromStore(ctx, visibility)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert class memo visibility")
		}
		visibilityMessages = append(visibilityMessages, visibilityMessage)
	}
	
	response := &v1pb.ListClassMemoVisibilitiesResponse{
		Visibilities:  visibilityMessages,
		NextPageToken: nextPageToken,
	}
	return response, nil
}

// CreateClassTagTemplate creates a tag template for a class.
func (s *APIV1Service) CreateClassTagTemplate(ctx context.Context, request *v1pb.CreateClassTagTemplateRequest) (*v1pb.ClassTagTemplate, error) {
	// Extract class UID from class resource name
	classUID, err := ExtractClassUIDFromName(request.Class)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class name: %v", err)
	}
	
	// Validate request
	if request.TagTemplate == nil {
		return nil, status.Errorf(codes.InvalidArgument, "tag_template is required")
	}
	if request.TagTemplate.DisplayName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "tag_template.display_name is required")
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{UID: &classUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check permissions: only class teachers/admins can create tag templates
	if !s.canManageClass(currentUser, class) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied: only class teachers and administrators can create tag templates")
	}
	
	// Check if tag template with same name already exists in this class
	existingTemplate, err := s.Store.GetClassTagTemplate(ctx, &store.FindClassTagTemplate{
		ClassID: &class.ID,
		Name:    &request.TagTemplate.DisplayName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check existing tag template: %v", err)
	}
	if existingTemplate != nil {
		return nil, status.Errorf(codes.AlreadyExists, "tag template with name %q already exists in this class", request.TagTemplate.DisplayName)
	}
	
	// Create tag template
	now := time.Now().Unix()
	color := ""
	if request.TagTemplate.Color != nil {
		color = *request.TagTemplate.Color
	}
	
	tagTemplate := &store.ClassTagTemplate{
		ClassID:     class.ID,
		Name:        request.TagTemplate.DisplayName,
		Color:       color,
		Description: request.TagTemplate.Description,
		CreatedTs:   now,
		UpdatedTs:   now,
	}
	
	createdTemplate, err := s.Store.CreateClassTagTemplate(ctx, tagTemplate)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return nil, status.Errorf(codes.AlreadyExists, "tag template already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create tag template: %v", err)
	}
	
	// Convert to protobuf response
	templateMessage, err := s.convertClassTagTemplateFromStore(ctx, createdTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert class tag template")
	}
	
	slog.Info("Class tag template created", 
		slog.String("class", class.UID), 
		slog.String("template_name", createdTemplate.Name))
	
	return templateMessage, nil
}

// UpdateClassTagTemplate updates a tag template.
func (s *APIV1Service) UpdateClassTagTemplate(ctx context.Context, request *v1pb.UpdateClassTagTemplateRequest) (*v1pb.ClassTagTemplate, error) {
	// Extract template ID from resource name
	templateID, err := ExtractClassTagTemplateIDFromName(request.TagTemplate.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class tag template name: %v", err)
	}
	
	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask is required")
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get tag template
	tagTemplate, err := s.Store.GetClassTagTemplate(ctx, &store.FindClassTagTemplate{ID: &templateID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class tag template: %v", err)
	}
	if tagTemplate == nil {
		return nil, status.Errorf(codes.NotFound, "class tag template not found")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{ID: &tagTemplate.ClassID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check permissions: only class teachers/admins can update tag templates
	if !s.canManageClass(currentUser, class) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied: only class teachers and administrators can update tag templates")
	}
	
	// Prepare update
	update := &store.UpdateClassTagTemplate{
		ID: tagTemplate.ID,
	}
	
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "display_name":
			if request.TagTemplate.DisplayName == "" {
				return nil, status.Errorf(codes.InvalidArgument, "tag_template.display_name cannot be empty")
			}
			update.Name = &request.TagTemplate.DisplayName
			
			// Check if new name already exists in class (excluding current template)
			if request.TagTemplate.DisplayName != tagTemplate.Name {
				existingTemplate, err := s.Store.GetClassTagTemplate(ctx, &store.FindClassTagTemplate{
					ClassID: &class.ID,
					Name:    &request.TagTemplate.DisplayName,
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to check existing tag template: %v", err)
				}
				if existingTemplate != nil && existingTemplate.ID != tagTemplate.ID {
					return nil, status.Errorf(codes.AlreadyExists, "tag template with name %q already exists in this class", request.TagTemplate.DisplayName)
				}
			}
			
		case "description":
			update.Description = &request.TagTemplate.Description
		case "color":
			if request.TagTemplate.Color != nil {
				color := *request.TagTemplate.Color
				update.Color = &color
			} else {
				// Clear color
				emptyString := ""
				update.Color = &emptyString
			}
		}
	}
	
	// Apply update
	if err = s.Store.UpdateClassTagTemplate(ctx, update); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update tag template: %v", err)
	}
	
	// Get updated template
	updatedTemplate, err := s.Store.GetClassTagTemplate(ctx, &store.FindClassTagTemplate{ID: &templateID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated tag template: %v", err)
	}
	if updatedTemplate == nil {
		return nil, status.Errorf(codes.NotFound, "updated tag template not found")
	}
	
	// Convert to protobuf response
	templateMessage, err := s.convertClassTagTemplateFromStore(ctx, updatedTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert class tag template")
	}
	
	slog.Info("Class tag template updated", 
		slog.String("class", class.UID), 
		slog.String("template_name", updatedTemplate.Name))
	
	return templateMessage, nil
}

// DeleteClassTagTemplate deletes a tag template.
func (s *APIV1Service) DeleteClassTagTemplate(ctx context.Context, request *v1pb.DeleteClassTagTemplateRequest) (*emptypb.Empty, error) {
	// Extract template ID from resource name
	templateID, err := ExtractClassTagTemplateIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class tag template name: %v", err)
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get tag template
	tagTemplate, err := s.Store.GetClassTagTemplate(ctx, &store.FindClassTagTemplate{ID: &templateID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class tag template: %v", err)
	}
	if tagTemplate == nil {
		return nil, status.Errorf(codes.NotFound, "class tag template not found")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{ID: &tagTemplate.ClassID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check permissions: only class teachers/admins can delete tag templates
	if !s.canManageClass(currentUser, class) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied: only class teachers and administrators can delete tag templates")
	}
	
	// Delete tag template
	if err = s.Store.DeleteClassTagTemplate(ctx, &store.DeleteClassTagTemplate{ID: tagTemplate.ID}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete tag template: %v", err)
	}
	
	slog.Info("Class tag template deleted", 
		slog.String("class", class.UID), 
		slog.String("template_name", tagTemplate.Name),
		slog.Int("template_id", int(tagTemplate.ID)))
	
	return &emptypb.Empty{}, nil
}

// ListClassTagTemplates lists tag templates for a class.
func (s *APIV1Service) ListClassTagTemplates(ctx context.Context, request *v1pb.ListClassTagTemplatesRequest) (*v1pb.ListClassTagTemplatesResponse, error) {
	// Extract class UID from class resource name
	classUID, err := ExtractClassUIDFromName(request.Class)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid class name: %v", err)
	}
	
	// Get current user
	currentUser, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	
	// Get class
	class, err := s.Store.GetClass(ctx, &store.FindClass{UID: &classUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get class: %v", err)
	}
	if class == nil {
		return nil, status.Errorf(codes.NotFound, "class not found")
	}
	
	// Check if user can view the class
	canView, err := s.canViewClass(ctx, currentUser, class)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check view permissions: %v", err)
	}
	if !canView {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied: cannot view this class")
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
	
	// Find tag templates
	templateFind := &store.FindClassTagTemplate{
		ClassID: &class.ID,
		Limit:   &limitPlusOne,
		Offset:  &offset,
	}
	
	templates, err := s.Store.ListClassTagTemplates(ctx, templateFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list class tag templates: %v", err)
	}
	
	// Convert to protobuf messages
	templateMessages := []*v1pb.ClassTagTemplate{}
	nextPageToken := ""
	if len(templates) == limitPlusOne {
		templates = templates[:limit]
		nextPageToken, err = getPageToken(limit, offset+limit)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
	}
	
	for _, template := range templates {
		templateMessage, err := s.convertClassTagTemplateFromStore(ctx, template)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert class tag template")
		}
		templateMessages = append(templateMessages, templateMessage)
	}
	
	response := &v1pb.ListClassTagTemplatesResponse{
		TagTemplates:  templateMessages,
		NextPageToken: nextPageToken,
	}
	return response, nil
}

// Helper functions for resource name parsing

// ExtractClassMemberIDFromName extracts class member ID from resource name.
// Format: classes/{class}/members/{class_member}
func ExtractClassMemberIDFromName(name string) (int32, error) {
	tokens, err := GetNameParentTokens(name, ClassNamePrefix, "members/")
	if err != nil {
		return 0, err
	}
	if len(tokens) != 2 {
		return 0, errors.Errorf("invalid class member name: expected 2 tokens, got %d", len(tokens))
	}
	classUID := tokens[0]
	memberIDStr := tokens[1]
	
	// Validate class UID format
	if !base.UIDMatcher.MatchString(classUID) {
		return 0, errors.Errorf("invalid class UID format: %s", classUID)
	}
	
	memberID, err := util.ConvertStringToInt32(memberIDStr)
	if err != nil {
		return 0, errors.Errorf("invalid class member ID %q", memberIDStr)
	}
	return memberID, nil
}

// ExtractClassMemoVisibilityIDFromName extracts class memo visibility ID from resource name.
// Format: classes/{class}/memoVisibility/{memo_visibility}
func ExtractClassMemoVisibilityIDFromName(name string) (int32, error) {
	tokens, err := GetNameParentTokens(name, ClassNamePrefix, "memoVisibility/")
	if err != nil {
		return 0, err
	}
	if len(tokens) != 2 {
		return 0, errors.Errorf("invalid class memo visibility name: expected 2 tokens, got %d", len(tokens))
	}
	classUID := tokens[0]
	visibilityIDStr := tokens[1]
	
	// Validate class UID format
	if !base.UIDMatcher.MatchString(classUID) {
		return 0, errors.Errorf("invalid class UID format: %s", classUID)
	}
	
	visibilityID, err := util.ConvertStringToInt32(visibilityIDStr)
	if err != nil {
		return 0, errors.Errorf("invalid class memo visibility ID %q", visibilityIDStr)
	}
	return visibilityID, nil
}

// ExtractClassTagTemplateIDFromName extracts class tag template ID from resource name.
// Format: classes/{class}/tagTemplates/{tag_template}
func ExtractClassTagTemplateIDFromName(name string) (int32, error) {
	tokens, err := GetNameParentTokens(name, ClassNamePrefix, "tagTemplates/")
	if err != nil {
		return 0, err
	}
	if len(tokens) != 2 {
		return 0, errors.Errorf("invalid class tag template name: expected 2 tokens, got %d", len(tokens))
	}
	classUID := tokens[0]
	templateIDStr := tokens[1]
	
	// Validate class UID format
	if !base.UIDMatcher.MatchString(classUID) {
		return 0, errors.Errorf("invalid class UID format: %s", classUID)
	}
	
	templateID, err := util.ConvertStringToInt32(templateIDStr)
	if err != nil {
		return 0, errors.Errorf("invalid class tag template ID %q", templateIDStr)
	}
	return templateID, nil
}

// convertClassMemberRoleToStore converts protobuf ClassMemberRole to store.ClassMemberRole.
func convertClassMemberRoleToStore(role v1pb.ClassMemberRole) (store.ClassMemberRole, error) {
	switch role {
	case v1pb.ClassMemberRole_TEACHER:
		return store.ClassMemberRoleTeacher, nil
	case v1pb.ClassMemberRole_ASSISTANT:
		return store.ClassMemberRoleAssistant, nil
	case v1pb.ClassMemberRole_STUDENT:
		return store.ClassMemberRoleStudent, nil
	case v1pb.ClassMemberRole_PARENT:
		return store.ClassMemberRoleParent, nil
	default:
		return "", errors.Errorf("invalid class member role: %v", role)
	}
}

// convertClassMemberRoleFromStore converts store.ClassMemberRole to protobuf ClassMemberRole.
func convertClassMemberRoleFromStore(role store.ClassMemberRole) v1pb.ClassMemberRole {
	switch role {
	case store.ClassMemberRoleTeacher:
		return v1pb.ClassMemberRole_TEACHER
	case store.ClassMemberRoleAssistant:
		return v1pb.ClassMemberRole_ASSISTANT
	case store.ClassMemberRoleStudent:
		return v1pb.ClassMemberRole_STUDENT
	case store.ClassMemberRoleParent:
		return v1pb.ClassMemberRole_PARENT
	default:
		return v1pb.ClassMemberRole_CLASS_MEMBER_ROLE_UNSPECIFIED
	}
}

// convertClassMemberFromStore converts a store.ClassMember to a v1pb.ClassMember.
func (s *APIV1Service) convertClassMemberFromStore(ctx context.Context, member *store.ClassMember) (*v1pb.ClassMember, error) {
	if member == nil {
		return nil, errors.New("class member is nil")
	}
	
	// Get class information
	class, err := s.Store.GetClass(ctx, &store.FindClass{ID: &member.ClassID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get class")
	}
	if class == nil {
		return nil, errors.Errorf("class not found for ID %d", member.ClassID)
	}
	
	// Get user information
	user, err := s.Store.GetUser(ctx, &store.FindUser{ID: &member.UserID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}
	if user == nil {
		return nil, errors.Errorf("user not found for ID %d", member.UserID)
	}
	userName := fmt.Sprintf("%s%d", UserNamePrefix, user.ID)
	
	// Get invited by user information if available
	var invitedByName *string
	if member.InvitedBy != nil {
		invitedByUser, err := s.Store.GetUser(ctx, &store.FindUser{ID: member.InvitedBy})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get invited by user")
		}
		if invitedByUser != nil {
			invitedBy := fmt.Sprintf("%s%d", UserNamePrefix, invitedByUser.ID)
			invitedByName = &invitedBy
		}
	}
	
	className := fmt.Sprintf("%s%s", ClassNamePrefix, class.UID)
	memberName := fmt.Sprintf("%s/members/%d", className, member.ID)
	
	// Convert role
	role := convertClassMemberRoleFromStore(member.Role)
	
	// Convert timestamp
	joinTime := timestamppb.New(time.Unix(member.JoinedTs, 0))
	
	return &v1pb.ClassMember{
		Name:       memberName,
		Class:      className,
		User:       userName,
		Role:       role,
		JoinTime:   joinTime,
		InvitedBy:  invitedByName,
	}, nil
}

// convertClassMemoVisibilityFromStore converts a store.ClassMemoVisibility to a v1pb.ClassMemoVisibility.
func (s *APIV1Service) convertClassMemoVisibilityFromStore(ctx context.Context, visibility *store.ClassMemoVisibility) (*v1pb.ClassMemoVisibility, error) {
	if visibility == nil {
		return nil, errors.New("class memo visibility is nil")
	}
	
	// Get class information
	class, err := s.Store.GetClass(ctx, &store.FindClass{ID: &visibility.ClassID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get class")
	}
	if class == nil {
		return nil, errors.Errorf("class not found for ID %d", visibility.ClassID)
	}
	
	// Get memo information
	memo, err := s.Store.GetMemo(ctx, &store.FindMemo{ID: &visibility.MemoID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get memo")
	}
	if memo == nil {
		return nil, errors.Errorf("memo not found for ID %d", visibility.MemoID)
	}
	
	// Get user who shared the memo
	sharedByUser, err := s.Store.GetUser(ctx, &store.FindUser{ID: &visibility.SharedBy})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get shared by user")
	}
	if sharedByUser == nil {
		return nil, errors.Errorf("user not found for ID %d", visibility.SharedBy)
	}
	
	className := fmt.Sprintf("%s%s", ClassNamePrefix, class.UID)
	memoName := fmt.Sprintf("%s%d", MemoNamePrefix, memo.ID)
	visibilityName := fmt.Sprintf("%s/memoVisibility/%d", className, visibility.ID)
	
	// Convert visibility
	vis := convertClassVisibilityFromStore(visibility.Visibility)
	
	return &v1pb.ClassMemoVisibility{
		Name:       visibilityName,
		Class:      className,
		Memo:       memoName,
		Visibility: vis,
		// Note: store.ClassMemoVisibility has Description field but protobuf doesn't
		// Note: store.ClassMemoVisibility has SharedTs but protobuf doesn't have share_time
	}, nil
}

// convertClassTagTemplateFromStore converts a store.ClassTagTemplate to a v1pb.ClassTagTemplate.
func (s *APIV1Service) convertClassTagTemplateFromStore(ctx context.Context, template *store.ClassTagTemplate) (*v1pb.ClassTagTemplate, error) {
	if template == nil {
		return nil, errors.New("class tag template is nil")
	}
	
	// Get class information
	class, err := s.Store.GetClass(ctx, &store.FindClass{ID: &template.ClassID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get class")
	}
	if class == nil {
		return nil, errors.Errorf("class not found for ID %d", template.ClassID)
	}
	
	className := fmt.Sprintf("%s%s", ClassNamePrefix, class.UID)
	templateName := fmt.Sprintf("%s/tagTemplates/%d", className, template.ID)
	
	return &v1pb.ClassTagTemplate{
		Name:        templateName,
		Class:       className,
		DisplayName: template.Name,
		Description: template.Description,
		Color:       &template.Color,
	}, nil
}

// generateInviteCode generates a random alphanumeric invite code.
func generateInviteCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	// Use math/rand with time seed (not cryptographically secure but sufficient for invite codes)
	// In production, consider using crypto/rand
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}
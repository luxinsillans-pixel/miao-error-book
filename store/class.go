package store

import (
	"context"
	"errors"

	"github.com/usememos/memos/internal/base"
	storepb "github.com/usememos/memos/proto/gen/store"
)

// ClassVisibility defines the visibility level of a class.
type ClassVisibility string

const (
	// ClassVisibilityPublic means anyone can view the class (but not necessarily join).
	ClassVisibilityPublic ClassVisibility = "PUBLIC"
	// ClassVisibilityProtected means only invited members can view.
	ClassVisibilityProtected ClassVisibility = "PROTECTED"
	// ClassVisibilityPrivate means only members can view (invite-only).
	ClassVisibilityPrivate ClassVisibility = "PRIVATE"
)

// ClassMemberRole defines the role of a member in a class.
type ClassMemberRole string

const (
	// ClassMemberRoleTeacher has full control over the class.
	ClassMemberRoleTeacher ClassMemberRole = "TEACHER"
	// ClassMemberRoleAssistant can manage students and content but not class settings.
	ClassMemberRoleAssistant ClassMemberRole = "ASSISTANT"
	// ClassMemberRoleStudent can view and submit content.
	ClassMemberRoleStudent ClassMemberRole = "STUDENT"
	// ClassMemberRoleParent can view child's progress (read-only).
	ClassMemberRoleParent ClassMemberRole = "PARENT"
)

// Class represents a class (e.g., a classroom for error book).
type Class struct {
	ID          int32
	UID         string            // Unique identifier (slug)
	Name        string
	Description string
	CreatorID   int32
	CreatedTs   int64
	UpdatedTs   int64
	Visibility  ClassVisibility
	InviteCode  string            // Optional invite code for joining
	Settings    *storepb.ClassSettings
}

// ClassMember represents membership of a user in a class.
type ClassMember struct {
	ID        int32
	ClassID   int32
	UserID    int32
	Role      ClassMemberRole
	JoinedTs  int64
	InvitedBy *int32 // User who invited this member
}

// ClassMemoVisibility controls visibility of memos within a class.
type ClassMemoVisibility struct {
	ID          int32
	ClassID     int32
	MemoID      int32
	Visibility  ClassVisibility // Override visibility for this memo in class context
	SharedBy    int32           // User who shared the memo
	SharedTs    int64
	Description string          // Optional note about why shared
}

// ClassTagTemplate defines tag templates available for a class.
type ClassTagTemplate struct {
	ID          int32
	ClassID     int32
	Name        string
	Color       string
	Description string
	CreatedTs   int64
	UpdatedTs   int64
}

// FindClass is used to filter classes.
type FindClass struct {
	ID        *int32
	UID       *string
	IDList    []int32
	UIDList   []string
	CreatorID *int32
	MemberID  *int32 // Filter classes where this user is a member
	Visibility *ClassVisibility
	InviteCode *string // Find by invite code
	Filters   []string // Advanced filter expressions
	Limit     *int
	Offset    *int
	OrderBy   string // e.g., "created_ts desc"
}

// UpdateClass is used to update a class.
type UpdateClass struct {
	ID          int32
	UID         *string
	Name        *string
	Description *string
	Visibility  *ClassVisibility
	InviteCode  *string
	Settings    *storepb.ClassSettings
}

// DeleteClass is used to delete a class.
type DeleteClass struct {
	ID int32
}

// FindClassMember filters class members.
type FindClassMember struct {
	ID         *int32
	ClassID    *int32
	UserID     *int32
	Role       *ClassMemberRole
	ClassIDList []int32
	UserIDList  []int32
	Limit      *int
	Offset     *int
}

// UpdateClassMember updates a member's role.
type UpdateClassMember struct {
	ID     int32
	Role   *ClassMemberRole
}

// DeleteClassMember removes a member from a class.
type DeleteClassMember struct {
	ID int32
}

// FindClassMemoVisibility filters memo visibility records.
type FindClassMemoVisibility struct {
	ID      *int32
	ClassID *int32
	MemoID  *int32
	UserID  *int32 // Filter by user who shared
	Limit   *int
	Offset  *int
}

// UpdateClassMemoVisibility updates a memo visibility record.
type UpdateClassMemoVisibility struct {
	ID         int32
	Visibility *ClassVisibility
	Description *string
}

// DeleteClassMemoVisibility deletes a memo visibility record.
type DeleteClassMemoVisibility struct {
	ID int32
}

// FindClassTagTemplate filters tag templates.
type FindClassTagTemplate struct {
	ID         *int32
	ClassID    *int32
	Name       *string
	ClassIDList []int32
	Limit      *int
	Offset     *int
}

// UpdateClassTagTemplate updates a tag template.
type UpdateClassTagTemplate struct {
	ID          int32
	Name        *string
	Color       *string
	Description *string
}

// DeleteClassTagTemplate deletes a tag template.
type DeleteClassTagTemplate struct {
	ID int32
}

// Store methods for Class
func (s *Store) CreateClass(ctx context.Context, create *Class) (*Class, error) {
	if create.UID == "" {
		return nil, errors.New("uid is required")
	}
	if !base.UIDMatcher.MatchString(create.UID) {
		return nil, errors.New("invalid uid format")
	}
	return s.driver.CreateClass(ctx, create)
}

func (s *Store) ListClasses(ctx context.Context, find *FindClass) ([]*Class, error) {
	return s.driver.ListClasses(ctx, find)
}

func (s *Store) GetClass(ctx context.Context, find *FindClass) (*Class, error) {
	list, err := s.ListClasses(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

func (s *Store) UpdateClass(ctx context.Context, update *UpdateClass) error {
	if update.UID != nil && !base.UIDMatcher.MatchString(*update.UID) {
		return errors.New("invalid uid")
	}
	return s.driver.UpdateClass(ctx, update)
}

func (s *Store) DeleteClass(ctx context.Context, delete *DeleteClass) error {
	// TODO: Consider cascade deletion of members, memo visibility, tag templates
	return s.driver.DeleteClass(ctx, delete)
}

// Store methods for ClassMember
func (s *Store) CreateClassMember(ctx context.Context, create *ClassMember) (*ClassMember, error) {
	return s.driver.CreateClassMember(ctx, create)
}

func (s *Store) ListClassMembers(ctx context.Context, find *FindClassMember) ([]*ClassMember, error) {
	return s.driver.ListClassMembers(ctx, find)
}

func (s *Store) GetClassMember(ctx context.Context, find *FindClassMember) (*ClassMember, error) {
	list, err := s.ListClassMembers(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

func (s *Store) UpdateClassMember(ctx context.Context, update *UpdateClassMember) error {
	return s.driver.UpdateClassMember(ctx, update)
}

func (s *Store) DeleteClassMember(ctx context.Context, delete *DeleteClassMember) error {
	return s.driver.DeleteClassMember(ctx, delete)
}

// Store methods for ClassMemoVisibility
func (s *Store) CreateClassMemoVisibility(ctx context.Context, create *ClassMemoVisibility) (*ClassMemoVisibility, error) {
	return s.driver.CreateClassMemoVisibility(ctx, create)
}

func (s *Store) ListClassMemoVisibilities(ctx context.Context, find *FindClassMemoVisibility) ([]*ClassMemoVisibility, error) {
	return s.driver.ListClassMemoVisibilities(ctx, find)
}

func (s *Store) GetClassMemoVisibility(ctx context.Context, find *FindClassMemoVisibility) (*ClassMemoVisibility, error) {
	list, err := s.ListClassMemoVisibilities(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

func (s *Store) UpdateClassMemoVisibility(ctx context.Context, update *UpdateClassMemoVisibility) error {
	return s.driver.UpdateClassMemoVisibility(ctx, update)
}

func (s *Store) DeleteClassMemoVisibility(ctx context.Context, delete *DeleteClassMemoVisibility) error {
	return s.driver.DeleteClassMemoVisibility(ctx, delete)
}

// Store methods for ClassTagTemplate
func (s *Store) CreateClassTagTemplate(ctx context.Context, create *ClassTagTemplate) (*ClassTagTemplate, error) {
	return s.driver.CreateClassTagTemplate(ctx, create)
}

func (s *Store) ListClassTagTemplates(ctx context.Context, find *FindClassTagTemplate) ([]*ClassTagTemplate, error) {
	return s.driver.ListClassTagTemplates(ctx, find)
}

func (s *Store) GetClassTagTemplate(ctx context.Context, find *FindClassTagTemplate) (*ClassTagTemplate, error) {
	list, err := s.ListClassTagTemplates(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

func (s *Store) UpdateClassTagTemplate(ctx context.Context, update *UpdateClassTagTemplate) error {
	return s.driver.UpdateClassTagTemplate(ctx, update)
}

func (s *Store) DeleteClassTagTemplate(ctx context.Context, delete *DeleteClassTagTemplate) error {
	return s.driver.DeleteClassTagTemplate(ctx, delete)
}
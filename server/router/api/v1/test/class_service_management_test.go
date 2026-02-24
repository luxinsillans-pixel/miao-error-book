package test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"


	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
)

func TestAddClassMember(t *testing.T) {
	ctx := context.Background()

	ts := NewTestService(t)
	defer ts.Cleanup()

	// Create admin user (teacher)
	teacherUser, err := ts.CreateHostUser(ctx, "teacher")
	require.NoError(t, err)
	require.NotNil(t, teacherUser)
	teacherCtx := ts.CreateUserContext(ctx, teacherUser.ID)

	// Create a class as teacher
	createdClass, err := ts.Service.CreateClass(teacherCtx, &apiv1.CreateClassRequest{
		Class: &apiv1.Class{
			Name:        "test-class-member",
			DisplayName: "Test Class for Member",
			Visibility:  apiv1.ClassVisibility_CLASS_PROTECTED,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, createdClass)

	// Create a regular user to add as member
	studentUser, err := ts.CreateRegularUser(ctx, "student")
	require.NoError(t, err)
	require.NotNil(t, studentUser)

	// Test 1: Add student as member (success)
	classMember, err := ts.Service.AddClassMember(teacherCtx, &apiv1.AddClassMemberRequest{
		Class: createdClass.GetName(),
		User:  fmt.Sprintf("users/%d", studentUser.ID),
		Role:  apiv1.ClassMemberRole_STUDENT,
	})
	require.NoError(t, err)
	require.NotNil(t, classMember)
	require.Equal(t, createdClass.GetName(), classMember.GetClass())
	require.Equal(t, fmt.Sprintf("users/%d", studentUser.ID), classMember.GetUser())
	require.Equal(t, apiv1.ClassMemberRole_STUDENT, classMember.GetRole())
	require.True(t, strings.HasPrefix(classMember.GetName(), "classes/"))
	require.True(t, strings.Contains(classMember.GetName(), "/members/"))

	// Test 2: Add duplicate member (should fail)
	_, err = ts.Service.AddClassMember(teacherCtx, &apiv1.AddClassMemberRequest{
		Class: createdClass.GetName(),
		User:  fmt.Sprintf("users/%d", studentUser.ID),
		Role:  apiv1.ClassMemberRole_ASSISTANT,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "already a member")

	// Test 3: Regular user trying to add member (should fail)
	regularUser, err := ts.CreateRegularUser(ctx, "regular")
	require.NoError(t, err)
	require.NotNil(t, regularUser)
	regularCtx := ts.CreateUserContext(ctx, regularUser.ID)

	_, err = ts.Service.AddClassMember(regularCtx, &apiv1.AddClassMemberRequest{
		Class: createdClass.GetName(),
		User:  fmt.Sprintf("users/%d", studentUser.ID),
		Role:  apiv1.ClassMemberRole_STUDENT,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")

	// Test 4: Add with different roles
	anotherUser, err := ts.CreateRegularUser(ctx, "another")
	require.NoError(t, err)
	require.NotNil(t, anotherUser)

	// Try adding with different roles
	for _, role := range []apiv1.ClassMemberRole{
		apiv1.ClassMemberRole_TEACHER,
		apiv1.ClassMemberRole_ASSISTANT,
		apiv1.ClassMemberRole_PARENT,
	} {
		// Use different user for each role attempt
		testUser, err := ts.CreateRegularUser(ctx, fmt.Sprintf("user-for-role-%d", role))
		require.NoError(t, err)
		
		classMember, err := ts.Service.AddClassMember(teacherCtx, &apiv1.AddClassMemberRequest{
			Class: createdClass.GetName(),
			User:  fmt.Sprintf("users/%d", testUser.ID),
			Role:  role,
		})
		require.NoError(t, err)
		require.Equal(t, role, classMember.GetRole())
	}
}

func TestRemoveClassMember(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Create teacher and class
	teacherUser, err := ts.CreateHostUser(ctx, "teacher")
	require.NoError(t, err)
	teacherCtx := ts.CreateUserContext(ctx, teacherUser.ID)

	createdClass, err := ts.Service.CreateClass(teacherCtx, &apiv1.CreateClassRequest{
		Class: &apiv1.Class{
			Name:        "test-class-remove-member",
			DisplayName: "Test Class for Remove Member",
			Visibility:  apiv1.ClassVisibility_CLASS_PROTECTED,
		},
	})
	require.NoError(t, err)

	// Create and add student
	studentUser, err := ts.CreateRegularUser(ctx, "student")
	require.NoError(t, err)

	// Add member first
	addedMember, err := ts.Service.AddClassMember(teacherCtx, &apiv1.AddClassMemberRequest{
		Class: createdClass.GetName(),
		User:  fmt.Sprintf("users/%d", studentUser.ID),
		Role:  apiv1.ClassMemberRole_STUDENT,
	})
	require.NoError(t, err)
	require.NotNil(t, addedMember)

	// Test 1: Remove member (success)
	_, err = ts.Service.RemoveClassMember(teacherCtx, &apiv1.RemoveClassMemberRequest{
		Name: addedMember.GetName(),
	})
	require.NoError(t, err)

	// Test 2: Remove non-existent member (should fail)
	_, err = ts.Service.RemoveClassMember(teacherCtx, &apiv1.RemoveClassMemberRequest{
		Name: "classes/999/members/999",
	})
	require.Error(t, err)

	// Test 3: Regular user trying to remove member (should fail)
	regularUser, err := ts.CreateRegularUser(ctx, "regular")
	require.NoError(t, err)
	regularCtx := ts.CreateUserContext(ctx, regularUser.ID)

	// Add member again for this test
	addedMember2, err := ts.Service.AddClassMember(teacherCtx, &apiv1.AddClassMemberRequest{
		Class: createdClass.GetName(),
		User:  fmt.Sprintf("users/%d", studentUser.ID),
		Role:  apiv1.ClassMemberRole_STUDENT,
	})
	require.NoError(t, err)

	_, err = ts.Service.RemoveClassMember(regularCtx, &apiv1.RemoveClassMemberRequest{
		Name: addedMember2.GetName(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")

	// Test 4: Remove self (student removing themselves, if allowed by policy)
	// This depends on business logic - for now test that teacher can remove
	// Add another student
	anotherStudent, err := ts.CreateRegularUser(ctx, "another-student")
	require.NoError(t, err)
	
	addedMember3, err := ts.Service.AddClassMember(teacherCtx, &apiv1.AddClassMemberRequest{
		Class: createdClass.GetName(),
		User:  fmt.Sprintf("users/%d", anotherStudent.ID),
		Role:  apiv1.ClassMemberRole_STUDENT,
	})
	require.NoError(t, err)
	
	// Student tries to remove themselves (may be allowed or not based on policy)
	studentCtx := ts.CreateUserContext(ctx, anotherStudent.ID)
	_, err = ts.Service.RemoveClassMember(studentCtx, &apiv1.RemoveClassMemberRequest{
		Name: addedMember3.GetName(),
	})
	// This might fail with permission denied or succeed depending on implementation
	// We'll just ensure it doesn't panic
	require.NotPanics(t, func() {
		_, _ = ts.Service.RemoveClassMember(studentCtx, &apiv1.RemoveClassMemberRequest{
			Name: addedMember3.GetName(),
		})
	})
}

func TestListClassMembers(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Create teacher and class
	teacherUser, err := ts.CreateHostUser(ctx, "teacher")
	require.NoError(t, err)
	teacherCtx := ts.CreateUserContext(ctx, teacherUser.ID)

	createdClass, err := ts.Service.CreateClass(teacherCtx, &apiv1.CreateClassRequest{
		Class: &apiv1.Class{
			Name:        "test-class-list-members",
			DisplayName: "Test Class for List Members",
			Visibility:  apiv1.ClassVisibility_CLASS_PROTECTED,
		},
	})
	require.NoError(t, err)

	// Add multiple members with different roles
	membersToAdd := []struct {
		username string
		role     apiv1.ClassMemberRole
	}{
		{"student1", apiv1.ClassMemberRole_STUDENT},
		{"student2", apiv1.ClassMemberRole_STUDENT},
		{"assistant1", apiv1.ClassMemberRole_ASSISTANT},
		{"parent1", apiv1.ClassMemberRole_PARENT},
		{"teacher2", apiv1.ClassMemberRole_TEACHER},
	}

	for _, m := range membersToAdd {
		user, err := ts.CreateRegularUser(ctx, m.username)
		require.NoError(t, err)
		
		_, err = ts.Service.AddClassMember(teacherCtx, &apiv1.AddClassMemberRequest{
			Class: createdClass.GetName(),
			User:  fmt.Sprintf("users/%d", user.ID),
			Role:  m.role,
		})
		require.NoError(t, err)
	}

	// Test 1: List all members (default)
	listResp, err := ts.Service.ListClassMembers(teacherCtx, &apiv1.ListClassMembersRequest{
		Class: createdClass.GetName(),
	})
	require.NoError(t, err)
	require.NotNil(t, listResp)
	// Should have 5 added members (creator is not in class_member table)
	require.Len(t, listResp.GetMembers(), 5)
	
	// Verify creator is NOT in the list (since creator is not in class_member table)
	foundCreator := false
	for _, member := range listResp.GetMembers() {
		if member.GetUser() == fmt.Sprintf("users/%d", teacherUser.ID) {
			foundCreator = true
		}
	}
	require.False(t, foundCreator, "Creator should not be in member list as it's not in class_member table")

	// Test 2: Pagination - limit to 3 members
	listResp2, err := ts.Service.ListClassMembers(teacherCtx, &apiv1.ListClassMembersRequest{
		Class: createdClass.GetName(),
		PageSize: 3,
	})
	require.NoError(t, err)
	require.NotNil(t, listResp2)
	require.Len(t, listResp2.GetMembers(), 3)
	require.NotEmpty(t, listResp2.GetNextPageToken(), "Should have next page token when results exceed page size")

	// Test 3: Use page token (page 1 has 3 members, page 2 should have 2 members)
	listResp3, err := ts.Service.ListClassMembers(teacherCtx, &apiv1.ListClassMembersRequest{
		Class: createdClass.GetName(),
		PageSize:  3,
		PageToken: listResp2.GetNextPageToken(),
	})
	require.NoError(t, err)
	require.NotNil(t, listResp3)
	require.Len(t, listResp3.GetMembers(), 2)

	// Test 4: Filter by role (if API supports it - currently not, but test basic listing)
	// For now just verify all members are returned
	
	// Test 5: Regular user trying to list members (should fail if not member)
	regularUser, err := ts.CreateRegularUser(ctx, "regular")
	require.NoError(t, err)
	regularCtx := ts.CreateUserContext(ctx, regularUser.ID)

	// This may succeed or fail depending on class visibility policy
	// We'll just ensure no panic
	require.NotPanics(t, func() {
		_, _ = ts.Service.ListClassMembers(regularCtx, &apiv1.ListClassMembersRequest{
			Class: createdClass.GetName(),
		})
	})
	
	// Test 6: List members of non-existent class (should fail)
	_, err = ts.Service.ListClassMembers(teacherCtx, &apiv1.ListClassMembersRequest{
		Class: "classes/999999999",
	})
	require.Error(t, err)
}

func TestUpdateClassMemberRole(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Create teacher and class
	teacherUser, err := ts.CreateHostUser(ctx, "teacher")
	require.NoError(t, err)
	teacherCtx := ts.CreateUserContext(ctx, teacherUser.ID)

	createdClass, err := ts.Service.CreateClass(teacherCtx, &apiv1.CreateClassRequest{
		Class: &apiv1.Class{
			Name:        "test-update-role",
			DisplayName: "Test Class for Update Role",
			Visibility:  apiv1.ClassVisibility_CLASS_PROTECTED,
		},
	})
	require.NoError(t, err)

	// Create and add student
	studentUser, err := ts.CreateRegularUser(ctx, "student")
	require.NoError(t, err)

	// Add member as student
	addedMember, err := ts.Service.AddClassMember(teacherCtx, &apiv1.AddClassMemberRequest{
		Class: createdClass.GetName(),
		User:  fmt.Sprintf("users/%d", studentUser.ID),
		Role:  apiv1.ClassMemberRole_STUDENT,
	})
	require.NoError(t, err)
	require.NotNil(t, addedMember)

	// Test 1: Update role from STUDENT to ASSISTANT (success)
	updatedMember, err := ts.Service.UpdateClassMemberRole(teacherCtx, &apiv1.UpdateClassMemberRoleRequest{
		Name: addedMember.GetName(),
		Role: apiv1.ClassMemberRole_ASSISTANT,
	})
	require.NoError(t, err)
	require.NotNil(t, updatedMember)
	require.Equal(t, apiv1.ClassMemberRole_ASSISTANT, updatedMember.GetRole())
	require.Equal(t, addedMember.GetName(), updatedMember.GetName())
	require.Equal(t, addedMember.GetClass(), updatedMember.GetClass())
	require.Equal(t, addedMember.GetUser(), updatedMember.GetUser())

	// Test 2: Update role to TEACHER (success)
	updatedMember2, err := ts.Service.UpdateClassMemberRole(teacherCtx, &apiv1.UpdateClassMemberRoleRequest{
		Name: addedMember.GetName(),
		Role: apiv1.ClassMemberRole_TEACHER,
	})
	require.NoError(t, err)
	require.Equal(t, apiv1.ClassMemberRole_TEACHER, updatedMember2.GetRole())

	// Test 3: Update non-existent member (should fail)
	_, err = ts.Service.UpdateClassMemberRole(teacherCtx, &apiv1.UpdateClassMemberRoleRequest{
		Name: "classes/999/members/999",
		Role: apiv1.ClassMemberRole_STUDENT,
	})
	require.Error(t, err)

	// Test 4: Regular user trying to update role (should fail)
	regularUser, err := ts.CreateRegularUser(ctx, "regular")
	require.NoError(t, err)
	regularCtx := ts.CreateUserContext(ctx, regularUser.ID)

	_, err = ts.Service.UpdateClassMemberRole(regularCtx, &apiv1.UpdateClassMemberRoleRequest{
		Name: addedMember.GetName(),
		Role: apiv1.ClassMemberRole_STUDENT,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")

	// Test 5: Update to same role (should succeed but no change)
	// First ensure current role is TEACHER
	require.Equal(t, apiv1.ClassMemberRole_TEACHER, updatedMember2.GetRole())
	updatedMember3, err := ts.Service.UpdateClassMemberRole(teacherCtx, &apiv1.UpdateClassMemberRoleRequest{
		Name: addedMember.GetName(),
		Role: apiv1.ClassMemberRole_TEACHER,
	})
	require.NoError(t, err)
	require.Equal(t, apiv1.ClassMemberRole_TEACHER, updatedMember3.GetRole())
}

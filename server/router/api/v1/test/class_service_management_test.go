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

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { classServiceClient } from "@/connect";
import { 
  GetClassRequest, 
  ListClassesRequest, 
  UpdateClassRequest, 
  DeleteClassRequest,
  AddClassMemberRequest,
  RemoveClassMemberRequest,
  ListClassMembersRequest,
  UpdateClassMemberRoleRequest,
  Class,
  ClassMember,
  ClassMemberRole,
  ClassVisibility,
  ClassTagTemplate,
  ClassMemoVisibility
} from "@/types/proto/api/v1/class_service_pb";
import { toast } from "sonner";

// Helper function to extract class ID from resource name
export const extractClassId = (name: string): string => {
  const parts = name.split('/');
  return parts[parts.length - 1];
};

// Helper function to extract member ID from resource name
export const extractMemberId = (name: string): string => {
  const parts = name.split('/');
  return parts[parts.length - 1];
};

// Class queries
export const useClass = (classId: string) => {
  return useQuery({
    queryKey: ["class", classId],
    queryFn: async () => {
      const request = new GetClassRequest();
      request.name = `classes/${classId}`;
      
      const response = await classServiceClient.getClass(request);
      return response;
    },
    enabled: !!classId,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
};

export const useClasses = (filter?: string, pageSize: number = 50) => {
  return useQuery({
    queryKey: ["classes", filter, pageSize],
    queryFn: async () => {
      const request = new ListClassesRequest();
      request.filter = filter || "";
      request.pageSize = pageSize;
      request.pageToken = "";
      
      const response = await classServiceClient.listClasses(request);
      return response;
    },
  });
};

export const useUpdateClass = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ classId, class: classData, updateMask }: { 
      classId: string; 
      class: Partial<Class>; 
      updateMask?: string[] 
    }) => {
      const request = new UpdateClassRequest();
      
      const fullClass = {
        ...classData,
        name: `classes/${classId}`,
      } as Class;
      
      request.class = fullClass;
      
      if (updateMask) {
        // In a real implementation, this would create a FieldMask
        // For now, we'll send the full class
      }
      
      const response = await classServiceClient.updateClass(request);
      return response;
    },
    onSuccess: (data, variables) => {
      toast.success("班级更新成功");
      queryClient.invalidateQueries({ queryKey: ["class", variables.classId] });
      queryClient.invalidateQueries({ queryKey: ["classes"] });
    },
    onError: (error) => {
      toast.error(`更新失败: ${error.message}`);
    },
  });
};

export const useDeleteClass = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (classId: string) => {
      const request = new DeleteClassRequest();
      request.name = `classes/${classId}`;
      
      await classServiceClient.deleteClass(request);
    },
    onSuccess: (_, classId) => {
      toast.success("班级删除成功");
      queryClient.invalidateQueries({ queryKey: ["classes"] });
      queryClient.removeQueries({ queryKey: ["class", classId] });
    },
    onError: (error) => {
      toast.error(`删除失败: ${error.message}`);
    },
  });
};

// Class member queries
export const useClassMembers = (classId: string, pageSize: number = 100) => {
  return useQuery({
    queryKey: ["classMembers", classId],
    queryFn: async () => {
      const request = new ListClassMembersRequest();
      request.class = `classes/${classId}`;
      request.pageSize = pageSize;
      request.pageToken = "";
      
      const response = await classServiceClient.listClassMembers(request);
      return response.members;
    },
    enabled: !!classId,
  });
};

export const useAddClassMember = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ classId, userId, role }: { 
      classId: string; 
      userId: string; 
      role: ClassMemberRole 
    }) => {
      const request = new AddClassMemberRequest();
      request.class = `classes/${classId}`;
      request.user = `users/${userId}`;
      request.role = role;
      
      const response = await classServiceClient.addClassMember(request);
      return response;
    },
    onSuccess: (_, variables) => {
      toast.success("成员添加成功");
      queryClient.invalidateQueries({ queryKey: ["classMembers", variables.classId] });
    },
    onError: (error) => {
      toast.error(`添加成员失败: ${error.message}`);
    },
  });
};

export const useRemoveClassMember = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ classId, memberId }: { 
      classId: string; 
      memberId: string 
    }) => {
      const request = new RemoveClassMemberRequest();
      request.name = `classes/${classId}/members/${memberId}`;
      
      await classServiceClient.removeClassMember(request);
    },
    onSuccess: (_, variables) => {
      toast.success("成员移除成功");
      queryClient.invalidateQueries({ queryKey: ["classMembers", variables.classId] });
    },
    onError: (error) => {
      toast.error(`移除成员失败: ${error.message}`);
    },
  });
};

export const useUpdateClassMemberRole = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ classId, memberId, role }: { 
      classId: string; 
      memberId: string; 
      role: ClassMemberRole 
    }) => {
      const request = new UpdateClassMemberRoleRequest();
      request.name = `classes/${classId}/members/${memberId}`;
      request.role = role;
      
      const response = await classServiceClient.updateClassMemberRole(request);
      return response;
    },
    onSuccess: (_, variables) => {
      toast.success("成员角色更新成功");
      queryClient.invalidateQueries({ queryKey: ["classMembers", variables.classId] });
    },
    onError: (error) => {
      toast.error(`更新角色失败: ${error.message}`);
    },
  });
};

// Class memo visibility queries
export const useClassMemoVisibilities = (classId: string, pageSize: number = 50) => {
  return useQuery({
    queryKey: ["classMemoVisibilities", classId],
    queryFn: async () => {
      const request = new ListClassMemoVisibilitiesRequest();
      request.class = `classes/${classId}`;
      request.pageSize = pageSize;
      request.pageToken = "";
      
      const response = await classServiceClient.listClassMemoVisibilities(request);
      return response.visibilities;
    },
    enabled: !!classId,
  });
};

// Class tag template queries
export const useClassTagTemplates = (classId: string, pageSize: number = 50) => {
  return useQuery({
    queryKey: ["classTagTemplates", classId],
    queryFn: async () => {
      const request = new ListClassTagTemplatesRequest();
      request.class = `classes/${classId}`;
      request.pageSize = pageSize;
      request.pageToken = "";
      
      const response = await classServiceClient.listClassTagTemplates(request);
      return response.tagTemplates;
    },
    enabled: !!classId,
  });
};

// Helper functions
export const getRoleDisplayName = (role: ClassMemberRole): string => {
  switch (role) {
    case ClassMemberRole.TEACHER:
      return "教师";
    case ClassMemberRole.ASSISTANT:
      return "助教";
    case ClassMemberRole.STUDENT:
      return "学生";
    case ClassMemberRole.PARENT:
      return "家长";
    default:
      return "未知";
  }
};

export const getRoleColor = (role: ClassMemberRole): string => {
  switch (role) {
    case ClassMemberRole.TEACHER:
      return "bg-purple-500 text-white";
    case ClassMemberRole.ASSISTANT:
      return "bg-blue-500 text-white";
    case ClassMemberRole.STUDENT:
      return "bg-green-500 text-white";
    case ClassMemberRole.PARENT:
      return "bg-yellow-500 text-white";
    default:
      return "bg-gray-500 text-white";
  }
};

export const getVisibilityDisplayName = (visibility: ClassVisibility): string => {
  switch (visibility) {
    case ClassVisibility.CLASS_PUBLIC:
      return "公开";
    case ClassVisibility.CLASS_PROTECTED:
      return "受保护";
    case ClassVisibility.CLASS_PRIVATE:
      return "私有";
    default:
      return "未知";
  }
};

export const getVisibilityColor = (visibility: ClassVisibility): string => {
  switch (visibility) {
    case ClassVisibility.CLASS_PUBLIC:
      return "bg-green-500 text-white";
    case ClassVisibility.CLASS_PROTECTED:
      return "bg-yellow-500 text-white";
    case ClassVisibility.CLASS_PRIVATE:
      return "bg-red-500 text-white";
    default:
      return "bg-gray-500 text-white";
  }
};
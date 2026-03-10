import { useState } from "react";
import { 
  UserPlus, 
  Search, 
  Filter,
  MoreVertical,
  Shield,
  UserCog,
  GraduationCap,
  UserMinus,
  Mail,
  Clock
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { 
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";

import { 
  useClassMembers, 
  useAddClassMember, 
  useRemoveClassMember, 
  useUpdateClassMemberRole,
  getRoleDisplayName,
  getRoleColor,
  extractMemberId
} from "@/hooks/useClassQueries";
import { ClassMemberRole } from "@/types/proto/api/v1/class_service_pb";
import AddMemberDialog from "./AddMemberDialog";

interface ClassMembersTabProps {
  classId: string;
}

const ClassMembersTab = ({ classId }: ClassMembersTabProps) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [isAddMemberDialogOpen, setIsAddMemberDialogOpen] = useState(false);
  const [selectedMember, setSelectedMember] = useState<string | null>(null);

  const { data: members, isLoading, refetch } = useClassMembers(classId);
  const addMemberMutation = useAddClassMember();
  const removeMemberMutation = useRemoveClassMember();
  const updateRoleMutation = useUpdateClassMemberRole();

  // 过滤成员
  const filteredMembers = members?.filter(member => {
    const matchesSearch = member.user.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesRole = roleFilter === "all" || 
      getRoleDisplayName(member.role).toLowerCase() === roleFilter.toLowerCase();
    
    return matchesSearch && matchesRole;
  });

  const getRoleIcon = (role: ClassMemberRole) => {
    switch (role) {
      case ClassMemberRole.TEACHER:
        return <Shield className="h-4 w-4" />;
      case ClassMemberRole.ASSISTANT:
        return <UserCog className="h-4 w-4" />;
      case ClassMemberRole.STUDENT:
        return <GraduationCap className="h-4 w-4" />;
      case ClassMemberRole.PARENT:
        return <Mail className="h-4 w-4" />;
      default:
        return <MoreVertical className="h-4 w-4" />;
    }
  };

  const handleRemoveMember = async (memberId: string) => {
    if (!window.confirm("确定要移除此成员吗？")) return;
    
    try {
      await removeMemberMutation.mutateAsync({ classId, memberId });
      toast.success("成员已移除");
    } catch (error) {
      toast.error("移除失败");
    }
  };

  const handleUpdateRole = async (memberId: string, newRole: ClassMemberRole) => {
    try {
      await updateRoleMutation.mutateAsync({ classId, memberId, role: newRole });
      toast.success("角色已更新");
    } catch (error) {
      toast.error("更新失败");
    }
  };

  const formatJoinTime = (timestamp?: { seconds: bigint; nanos: number }) => {
    if (!timestamp) return "未知时间";
    const date = new Date(Number(timestamp.seconds) * 1000);
    return date.toLocaleDateString("zh-CN", {
      year: "numeric",
      month: "short",
      day: "numeric"
    });
  };

  const getRoleOptions = () => {
    return [
      { value: ClassMemberRole.TEACHER, label: "教师" },
      { value: ClassMemberRole.ASSISTANT, label: "助教" },
      { value: ClassMemberRole.STUDENT, label: "学生" },
      { value: ClassMemberRole.PARENT, label: "家长" },
    ];
  };

  return (
    <div className="space-y-6">
      {/* 头部 */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold">班级成员</h2>
          <p className="text-muted-foreground mt-1">
            管理班级成员和角色分配
          </p>
        </div>
        <Button onClick={() => setIsAddMemberDialogOpen(true)}>
          <UserPlus className="mr-2 h-4 w-4" />
          添加成员
        </Button>
      </div>

      {/* 搜索和筛选 */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="搜索成员..."
                className="pl-10"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <Select value={roleFilter} onValueChange={setRoleFilter}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="筛选角色" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部角色</SelectItem>
                  <SelectItem value="教师">教师</SelectItem>
                  <SelectItem value="助教">助教</SelectItem>
                  <SelectItem value="学生">学生</SelectItem>
                  <SelectItem value="家长">家长</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 成员列表 */}
      <Card>
        <CardHeader>
          <CardTitle>成员列表</CardTitle>
          <CardDescription>
            共 {filteredMembers?.length || 0} 名成员
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="flex items-center justify-between p-4 border rounded-lg">
                  <div className="flex items-center space-x-4">
                    <Skeleton className="h-10 w-10 rounded-full" />
                    <div>
                      <Skeleton className="h-4 w-32 mb-2" />
                      <Skeleton className="h-3 w-24" />
                    </div>
                  </div>
                  <Skeleton className="h-8 w-24" />
                </div>
              ))}
            </div>
          ) : filteredMembers?.length === 0 ? (
            <div className="text-center py-12">
              <div className="mx-auto w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-4">
                <Users className="h-6 w-6 text-muted-foreground" />
              </div>
              <h3 className="text-lg font-semibold mb-2">暂无成员</h3>
              <p className="text-muted-foreground mb-4">
                {searchTerm || roleFilter !== "all" 
                  ? "没有找到匹配的成员" 
                  : "当前班级还没有成员"}
              </p>
              <Button onClick={() => setIsAddMemberDialogOpen(true)}>
                <UserPlus className="mr-2 h-4 w-4" />
                添加第一个成员
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              {filteredMembers?.map((member) => (
                <div 
                  key={member.name} 
                  className="flex flex-col sm:flex-row sm:items-center justify-between p-4 border rounded-lg hover:bg-muted/50 transition-colors"
                >
                  <div className="flex items-start sm:items-center space-x-4 mb-4 sm:mb-0">
                    <div className="relative">
                      <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
                        <span className="font-semibold text-primary">
                          {member.user.replace("users/", "").charAt(0).toUpperCase()}
                        </span>
                      </div>
                      <div className="absolute -bottom-1 -right-1">
                        {getRoleIcon(member.role)}
                      </div>
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h4 className="font-semibold">{member.user.replace("users/", "")}</h4>
                        <Badge className={getRoleColor(member.role)}>
                          {getRoleDisplayName(member.role)}
                        </Badge>
                      </div>
                      <div className="flex items-center text-sm text-muted-foreground mt-1">
                        <Clock className="mr-1 h-3 w-3" />
                        加入时间: {formatJoinTime(member.joinTime)}
                        {member.invitedBy && (
                          <>
                            <span className="mx-2">•</span>
                            邀请人: {member.invitedBy.replace("users/", "")}
                          </>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <Select
                      value={member.role.toString()}
                      onValueChange={(value) => handleUpdateRole(
                        extractMemberId(member.name),
                        parseInt(value) as ClassMemberRole
                      )}
                    >
                      <SelectTrigger className="w-[120px]">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {getRoleOptions().map((option) => (
                          <SelectItem 
                            key={option.value} 
                            value={option.value.toString()}
                          >
                            {option.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>

                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm">
                          <MoreVertical className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuLabel>成员操作</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => handleRemoveMember(extractMemberId(member.name))}
                          className="text-red-600 focus:text-red-600"
                        >
                          <UserMinus className="mr-2 h-4 w-4" />
                          移除成员
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* 角色说明 */}
      <Card>
        <CardHeader>
          <CardTitle>角色说明</CardTitle>
          <CardDescription>不同角色的权限说明</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="border rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <Shield className="h-5 w-5 text-purple-500" />
                <h4 className="font-semibold">教师</h4>
              </div>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• 管理班级设置</li>
                <li>• 添加/移除成员</li>
                <li>• 管理所有笔记</li>
                <li>• 创建标签模板</li>
              </ul>
            </div>
            
            <div className="border rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <UserCog className="h-5 w-5 text-blue-500" />
                <h4 className="font-semibold">助教</h4>
              </div>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• 管理学生成员</li>
                <li>• 查看所有笔记</li>
                <li>• 批改学生作业</li>
                <li>• 管理标签模板</li>
              </ul>
            </div>
            
            <div className="border rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <GraduationCap className="h-5 w-5 text-green-500" />
                <h4 className="font-semibold">学生</h4>
              </div>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• 查看班级笔记</li>
                <li>• 提交个人笔记</li>
                <li>• 参与讨论</li>
                <li>• 使用标签模板</li>
              </ul>
            </div>
            
            <div className="border rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <Mail className="h-5 w-5 text-yellow-500" />
                <h4 className="font-semibold">家长</h4>
              </div>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• 查看班级动态</li>
                <li>• 查看学生笔记</li>
                <li>• 接收通知</li>
                <li>• 有限参与</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 添加成员对话框 */}
      <AddMemberDialog
        open={isAddMemberDialogOpen}
        onOpenChange={setIsAddMemberDialogOpen}
        classId={classId}
        onSuccess={() => {
          refetch();
          setIsAddMemberDialogOpen(false);
        }}
      />
    </div>
  );
};

export default ClassMembersTab;
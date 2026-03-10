import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { 
  ArrowLeft, 
  Users, 
  BookOpen, 
  Settings, 
  Tag, 
  Eye, 
  Calendar,
  Copy,
  Share2,
  MoreVertical,
  Edit,
  Trash2
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Separator } from "@/components/ui/separator";
import { 
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";

import { useClass, useDeleteClass, getVisibilityDisplayName, getVisibilityColor } from "@/hooks/useClassQueries";
import ClassMembersTab from "@/components/classes/ClassMembersTab";
import ClassSettingsTab from "@/components/classes/ClassSettingsTab";
import ClassMemosTab from "@/components/classes/ClassMemosTab";
import ClassTagTemplatesTab from "@/components/classes/ClassTagTemplatesTab";

const ClassDetail = () => {
  const { classId } = useParams<{ classId: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState("overview");
  const [isDeleting, setIsDeleting] = useState(false);

  const { 
    data: classData, 
    isLoading, 
    error 
  } = useClass(classId || "");

  const deleteClassMutation = useDeleteClass();

  if (error) {
    toast.error("加载班级失败");
    navigate("/classes");
    return null;
  }

  const handleCopyInviteCode = () => {
    if (classData?.inviteCode) {
      navigator.clipboard.writeText(classData.inviteCode);
      toast.success("邀请码已复制到剪贴板");
    } else {
      toast.error("该班级没有邀请码");
    }
  };

  const handleShareClass = () => {
    const classUrl = `${window.location.origin}/classes/${classId}`;
    navigator.clipboard.writeText(classUrl);
    toast.success("班级链接已复制到剪贴板");
  };

  const handleEditClass = () => {
    navigate(`/classes/${classId}/edit`);
  };

  const handleDeleteClass = async () => {
    if (!classId || !classData) return;
    
    if (window.confirm(`确定要删除班级 "${classData.displayName}" 吗？此操作不可撤销。`)) {
      setIsDeleting(true);
      try {
        await deleteClassMutation.mutateAsync(classId);
        toast.success("班级已删除");
        navigate("/classes");
      } catch (error) {
        toast.error("删除失败");
      } finally {
        setIsDeleting(false);
      }
    }
  };

  if (isLoading || !classData) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex flex-col space-y-6">
          {/* 头部骨架屏 */}
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Skeleton className="h-10 w-10 rounded" />
              <div>
                <Skeleton className="h-6 w-48 mb-2" />
                <Skeleton className="h-4 w-32" />
              </div>
            </div>
            <Skeleton className="h-10 w-24" />
          </div>

          {/* 标签页骨架屏 */}
          <Skeleton className="h-10 w-full" />
          
          {/* 内容骨架屏 */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-2">
              <Skeleton className="h-64 w-full" />
            </div>
            <div>
              <Skeleton className="h-64 w-full" />
            </div>
          </div>
        </div>
      </div>
    );
  }

  const formatDate = (timestamp?: { seconds: bigint; nanos: number }) => {
    if (!timestamp) return "未知时间";
    const date = new Date(Number(timestamp.seconds) * 1000);
    return date.toLocaleDateString("zh-CN", {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit"
    });
  };

  return (
    <div className="container mx-auto p-4 md:p-6">
      {/* 返回按钮和头部 */}
      <div className="mb-6">
        <Button
          variant="ghost"
          size="sm"
          className="mb-4"
          onClick={() => navigate("/classes")}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          返回班级列表
        </Button>

        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="flex-1">
            <div className="flex items-center gap-3">
              <div className="bg-primary/10 p-3 rounded-lg">
                <BookOpen className="h-6 w-6 text-primary" />
              </div>
              <div>
                <div className="flex items-center gap-3">
                  <h1 className="text-3xl font-bold">{classData.displayName}</h1>
                  <Badge className={getVisibilityColor(classData.visibility)}>
                    {getVisibilityDisplayName(classData.visibility)}
                  </Badge>
                </div>
                <p className="text-muted-foreground mt-1">
                  {classData.uid} • 创建者: {classData.creator.replace("users/", "")}
                </p>
              </div>
            </div>
            {classData.description && (
              <p className="text-muted-foreground mt-3 max-w-3xl">
                {classData.description}
              </p>
            )}
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleShareClass}
            >
              <Share2 className="mr-2 h-4 w-4" />
              分享
            </Button>
            
            {classData.inviteCode && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleCopyInviteCode}
              >
                <Copy className="mr-2 h-4 w-4" />
                邀请码
              </Button>
            )}

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm">
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>班级操作</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={handleEditClass}>
                  <Edit className="mr-2 h-4 w-4" />
                  编辑班级
                </DropdownMenuItem>
                <DropdownMenuItem 
                  onClick={handleDeleteClass}
                  className="text-red-600 focus:text-red-600"
                  disabled={isDeleting}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  {isDeleting ? "删除中..." : "删除班级"}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>

      {/* 标签页导航 */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid grid-cols-5 md:w-auto">
          <TabsTrigger value="overview">
            <Eye className="mr-2 h-4 w-4" />
            概览
          </TabsTrigger>
          <TabsTrigger value="members">
            <Users className="mr-2 h-4 w-4" />
            成员
          </TabsTrigger>
          <TabsTrigger value="memos">
            <BookOpen className="mr-2 h-4 w-4" />
            备忘录
          </TabsTrigger>
          <TabsTrigger value="tags">
            <Tag className="mr-2 h-4 w-4" />
            标签模板
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings className="mr-2 h-4 w-4" />
            设置
          </TabsTrigger>
        </TabsList>

        <Separator className="my-6" />

        {/* 概览标签页 */}
        <TabsContent value="overview" className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* 左侧：班级信息 */}
            <div className="lg:col-span-2 space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle>班级信息</CardTitle>
                  <CardDescription>班级详细信息和设置</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">班级ID</p>
                      <p className="font-mono">{classData.uid}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">可见性</p>
                      <Badge className={getVisibilityColor(classData.visibility)}>
                        {getVisibilityDisplayName(classData.visibility)}
                      </Badge>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">创建时间</p>
                      <p>{formatDate(classData.createTime)}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">更新时间</p>
                      <p>{formatDate(classData.updateTime)}</p>
                    </div>
                  </div>

                  {classData.settings && (
                    <>
                      <Separator />
                      <div>
                        <h3 className="font-semibold mb-2">班级设置</h3>
                        <div className="grid grid-cols-2 gap-4">
                          <div className="flex items-center space-x-2">
                            <div className={`h-3 w-3 rounded-full ${classData.settings.studentMemoVisibility ? 'bg-green-500' : 'bg-red-500'}`} />
                            <span className="text-sm">
                              学生笔记可见: {classData.settings.studentMemoVisibility ? '是' : '否'}
                            </span>
                          </div>
                          <div className="flex items-center space-x-2">
                            <div className={`h-3 w-3 rounded-full ${classData.settings.allowAnonymous ? 'bg-green-500' : 'bg-red-500'}`} />
                            <span className="text-sm">
                              允许匿名: {classData.settings.allowAnonymous ? '是' : '否'}
                            </span>
                          </div>
                          <div className="flex items-center space-x-2">
                            <div className={`h-3 w-3 rounded-full ${classData.settings.enableTagTemplates ? 'bg-green-500' : 'bg-red-500'}`} />
                            <span className="text-sm">
                              标签模板: {classData.settings.enableTagTemplates ? '启用' : '禁用'}
                            </span>
                          </div>
                          <div className="flex items-center space-x-2">
                            <span className="text-sm">
                              最大成员数: {classData.settings.maxMembers === 0 ? '无限制' : classData.settings.maxMembers}
                            </span>
                          </div>
                        </div>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>

              {/* 快速操作卡片 */}
              <Card>
                <CardHeader>
                  <CardTitle>快速操作</CardTitle>
                  <CardDescription>管理班级的快捷方式</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <Button 
                      variant="outline" 
                      className="flex flex-col h-auto py-4"
                      onClick={() => setActiveTab("members")}
                    >
                      <Users className="h-6 w-6 mb-2" />
                      <span>管理成员</span>
                    </Button>
                    <Button 
                      variant="outline" 
                      className="flex flex-col h-auto py-4"
                      onClick={() => setActiveTab("memos")}
                    >
                      <BookOpen className="h-6 w-6 mb-2" />
                      <span>查看笔记</span>
                    </Button>
                    <Button 
                      variant="outline" 
                      className="flex flex-col h-auto py-4"
                      onClick={() => setActiveTab("tags")}
                    >
                      <Tag className="h-6 w-6 mb-2" />
                      <span>标签模板</span>
                    </Button>
                    <Button 
                      variant="outline" 
                      className="flex flex-col h-auto py-4"
                      onClick={() => setActiveTab("settings")}
                    >
                      <Settings className="h-6 w-6 mb-2" />
                      <span>班级设置</span>
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* 右侧：统计信息 */}
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle>班级统计</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <Users className="mr-2 h-4 w-4 text-muted-foreground" />
                      <span>成员数量</span>
                    </div>
                    <span className="font-semibold">加载中...</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <BookOpen className="mr-2 h-4 w-4 text-muted-foreground" />
                      <span>笔记数量</span>
                    </div>
                    <span className="font-semibold">加载中...</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <Tag className="mr-2 h-4 w-4 text-muted-foreground" />
                      <span>标签模板</span>
                    </div>
                    <span className="font-semibold">加载中...</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <Calendar className="mr-2 h-4 w-4 text-muted-foreground" />
                      <span>活跃天数</span>
                    </div>
                    <span className="font-semibold">计算中...</span>
                  </div>
                </CardContent>
              </Card>

              {/* 邀请卡片 */}
              {classData.inviteCode && (
                <Card>
                  <CardHeader>
                    <CardTitle>邀请成员</CardTitle>
                    <CardDescription>使用邀请码添加新成员</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      <div className="bg-muted p-3 rounded-lg">
                        <p className="text-sm font-medium mb-1">邀请码</p>
                        <div className="flex items-center justify-between">
                          <code className="font-mono text-lg">{classData.inviteCode}</code>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={handleCopyInviteCode}
                          >
                            <Copy className="h-4 w-4" />
                          </Button>
                        </div>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        将此邀请码分享给想要加入班级的用户。他们可以使用此代码加入班级。
                      </p>
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          </div>
        </TabsContent>

        {/* 成员标签页 */}
        <TabsContent value="members">
          <ClassMembersTab classId={classId!} />
        </TabsContent>

        {/* 备忘录标签页 */}
        <TabsContent value="memos">
          <ClassMemosTab classId={classId!} />
        </TabsContent>

        {/* 标签模板标签页 */}
        <TabsContent value="tags">
          <ClassTagTemplatesTab classId={classId!} />
        </TabsContent>

        {/* 设置标签页 */}
        <TabsContent value="settings">
          <ClassSettingsTab classId={classId!} classData={classData} />
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default ClassDetail;
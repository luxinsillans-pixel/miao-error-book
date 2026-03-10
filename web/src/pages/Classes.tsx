import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Plus, Search, Users, BookOpen, Settings } from "lucide-react";
import { classServiceClient } from "@/connect";
import { ListClassesRequest } from "@/types/proto/api/v1/class_service_pb";
import { ClassVisibility } from "@/types/proto/api/v1/class_service_pb";
import CreateClassDialog from "@/components/classes/CreateClassDialog";

const ClassesPage = () => {
  const [searchTerm, setSearchTerm] = useState("");
  const [activeTab, setActiveTab] = useState("all");
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);

  // 查询班级列表
  const { data: classesData, isLoading, refetch } = useQuery({
    queryKey: ["classes", activeTab, searchTerm],
    queryFn: async () => {
      const request = new ListClassesRequest();
      request.filter = searchTerm;
      request.pageSize = 50;
      request.pageToken = "";
      
      const response = await classServiceClient.listClasses(request);
      return response.classes;
    },
  });

  const getVisibilityBadge = (visibility: ClassVisibility) => {
    switch (visibility) {
      case ClassVisibility.CLASS_PUBLIC:
        return <Badge variant="default" className="bg-green-500">公开</Badge>;
      case ClassVisibility.CLASS_PROTECTED:
        return <Badge variant="default" className="bg-yellow-500">受保护</Badge>;
      case ClassVisibility.CLASS_PRIVATE:
        return <Badge variant="default" className="bg-red-500">私有</Badge>;
      default:
        return <Badge variant="secondary">未知</Badge>;
    }
  };

  const filteredClasses = classesData?.filter(cls => {
    if (activeTab === "all") return true;
    if (activeTab === "public") return cls.visibility === ClassVisibility.CLASS_PUBLIC;
    if (activeTab === "protected") return cls.visibility === ClassVisibility.CLASS_PROTECTED;
    if (activeTab === "private") return cls.visibility === ClassVisibility.CLASS_PRIVATE;
    return true;
  });

  return (
    <div className="container mx-auto p-6">
      <div className="flex flex-col space-y-6">
        {/* 头部 */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">班级管理</h1>
            <p className="text-muted-foreground mt-2">
              管理您的班级，分享错误笔记和学习资源
            </p>
          </div>
          <Button onClick={() => setIsCreateDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            创建班级
          </Button>
        </div>

        {/* 搜索和筛选 */}
        <div className="flex flex-col space-y-4 md:flex-row md:items-center md:justify-between md:space-y-0">
          <div className="relative w-full md:w-1/3">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="搜索班级..."
              className="pl-10"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
          
          <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full md:w-auto">
            <TabsList>
              <TabsTrigger value="all">全部</TabsTrigger>
              <TabsTrigger value="public">公开</TabsTrigger>
              <TabsTrigger value="protected">受保护</TabsTrigger>
              <TabsTrigger value="private">私有</TabsTrigger>
            </TabsList>
          </Tabs>
        </div>

        {/* 班级列表 */}
        {isLoading ? (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <Card key={i} className="animate-pulse">
                <CardHeader>
                  <div className="h-6 bg-muted rounded w-3/4 mb-2"></div>
                  <div className="h-4 bg-muted rounded w-1/2"></div>
                </CardHeader>
                <CardContent>
                  <div className="h-4 bg-muted rounded w-full mb-2"></div>
                  <div className="h-4 bg-muted rounded w-2/3"></div>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : filteredClasses?.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-12">
              <BookOpen className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold">暂无班级</h3>
              <p className="text-muted-foreground text-center mt-2 mb-4">
                {searchTerm ? "没有找到匹配的班级" : "您还没有创建或加入任何班级"}
              </p>
              <Button onClick={() => setIsCreateDialogOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                创建第一个班级
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filteredClasses?.map((cls) => (
              <Card key={cls.name} className="hover:shadow-lg transition-shadow">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div>
                      <CardTitle className="text-lg">{cls.displayName}</CardTitle>
                      <CardDescription className="mt-1">
                        {cls.uid}
                      </CardDescription>
                    </div>
                    {getVisibilityBadge(cls.visibility)}
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground line-clamp-2">
                    {cls.description || "暂无描述"}
                  </p>
                  <div className="flex items-center mt-4 space-x-4 text-sm text-muted-foreground">
                    <div className="flex items-center">
                      <Users className="mr-1 h-4 w-4" />
                      <span>成员</span>
                    </div>
                    <div className="flex items-center">
                      <BookOpen className="mr-1 h-4 w-4" />
                      <span>笔记</span>
                    </div>
                  </div>
                </CardContent>
                <CardFooter className="flex justify-between">
                  <Button variant="outline" size="sm">查看详情</Button>
                  <Button variant="ghost" size="sm">
                    <Settings className="h-4 w-4" />
                  </Button>
                </CardFooter>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* 创建班级对话框 */}
      <CreateClassDialog
        open={isCreateDialogOpen}
        onOpenChange={setIsCreateDialogOpen}
        onSuccess={() => {
          refetch();
          setIsCreateDialogOpen(false);
        }}
      />
    </div>
  );
};

export default ClassesPage;
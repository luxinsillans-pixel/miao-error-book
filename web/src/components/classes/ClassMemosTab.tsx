import { useState } from "react";
import { Search, Filter, Eye, Lock, Globe, Calendar, User } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useClassMemoVisibilities } from "@/hooks/useClassQueries";
import { ClassVisibility } from "@/types/proto/api/v1/class_service_pb";

interface ClassMemosTabProps {
  classId: string;
}

const ClassMemosTab = ({ classId }: ClassMemosTabProps) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [visibilityFilter, setVisibilityFilter] = useState<string>("all");

  const { data: visibilities, isLoading } = useClassMemoVisibilities(classId);

  // 模拟备忘录数据（实际应该从API获取）
  const mockMemos = [
    {
      id: "1",
      title: "数据结构复习笔记",
      content: "二叉树、图、排序算法等重要概念总结...",
      author: "张三",
      visibility: ClassVisibility.CLASS_PUBLIC,
      createdAt: "2024-03-10",
      tags: ["数据结构", "复习"],
    },
    {
      id: "2",
      title: "操作系统实验报告",
      content: "进程调度和内存管理实验的详细分析...",
      author: "李四",
      visibility: ClassVisibility.CLASS_PROTECTED,
      createdAt: "2024-03-09",
      tags: ["操作系统", "实验"],
    },
    {
      id: "3",
      title: "计算机网络问题集",
      content: "TCP/IP协议栈常见问题和解决方案...",
      author: "王五",
      visibility: ClassVisibility.CLASS_PRIVATE,
      createdAt: "2024-03-08",
      tags: ["网络", "问题"],
    },
  ];

  const filteredMemos = mockMemos.filter(memo => {
    const matchesSearch = memo.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         memo.content.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         memo.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase()));
    
    const matchesVisibility = visibilityFilter === "all" || 
      memo.visibility.toString() === visibilityFilter;
    
    return matchesSearch && matchesVisibility;
  });

  const getVisibilityIcon = (visibility: ClassVisibility) => {
    switch (visibility) {
      case ClassVisibility.CLASS_PUBLIC:
        return <Globe className="h-4 w-4" />;
      case ClassVisibility.CLASS_PROTECTED:
        return <Eye className="h-4 w-4" />;
      case ClassVisibility.CLASS_PRIVATE:
        return <Lock className="h-4 w-4" />;
      default:
        return <Eye className="h-4 w-4" />;
    }
  };

  const getVisibilityText = (visibility: ClassVisibility) => {
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

  const getVisibilityColor = (visibility: ClassVisibility) => {
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

  return (
    <div className="space-y-6">
      {/* 头部 */}
      <div>
        <h2 className="text-2xl font-bold">班级备忘录</h2>
        <p className="text-muted-foreground mt-1">
          查看和管理班级中的笔记和资源
        </p>
      </div>

      {/* 搜索和筛选 */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="搜索备忘录..."
                className="pl-10"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <Select value={visibilityFilter} onValueChange={setVisibilityFilter}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="筛选可见性" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部可见性</SelectItem>
                  <SelectItem value={ClassVisibility.CLASS_PUBLIC.toString()}>公开</SelectItem>
                  <SelectItem value={ClassVisibility.CLASS_PROTECTED.toString()}>受保护</SelectItem>
                  <SelectItem value={ClassVisibility.CLASS_PRIVATE.toString()}>私有</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 备忘录列表 */}
      <Card>
        <CardHeader>
          <CardTitle>备忘录列表</CardTitle>
          <CardDescription>
            共 {filteredMemos.length} 条备忘录
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="p-4 border rounded-lg">
                  <Skeleton className="h-5 w-1/2 mb-2" />
                  <Skeleton className="h-4 w-full mb-2" />
                  <Skeleton className="h-4 w-2/3" />
                </div>
              ))}
            </div>
          ) : filteredMemos.length === 0 ? (
            <div className="text-center py-12">
              <div className="mx-auto w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-4">
                <Search className="h-6 w-6 text-muted-foreground" />
              </div>
              <h3 className="text-lg font-semibold mb-2">暂无备忘录</h3>
              <p className="text-muted-foreground mb-4">
                {searchTerm || visibilityFilter !== "all" 
                  ? "没有找到匹配的备忘录" 
                  : "当前班级还没有备忘录"}
              </p>
              <Button>创建第一个备忘录</Button>
            </div>
          ) : (
            <div className="space-y-4">
              {filteredMemos.map((memo) => (
                <Link 
                  key={memo.id} 
                  to={`/memos/${memo.id}`}
                  className="block"
                >
                  <Card className="hover:shadow-lg transition-shadow cursor-pointer">
                    <CardContent className="pt-6">
                      <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-2">
                            <h3 className="font-semibold text-lg">{memo.title}</h3>
                            <Badge className={getVisibilityColor(memo.visibility)}>
                              <div className="flex items-center gap-1">
                                {getVisibilityIcon(memo.visibility)}
                                {getVisibilityText(memo.visibility)}
                              </div>
                            </Badge>
                          </div>
                          
                          <p className="text-muted-foreground mb-3 line-clamp-2">
                            {memo.content}
                          </p>
                          
                          <div className="flex flex-wrap gap-2 mb-3">
                            {memo.tags.map((tag) => (
                              <Badge key={tag} variant="outline">
                                {tag}
                              </Badge>
                            ))}
                          </div>
                          
                          <div className="flex items-center text-sm text-muted-foreground">
                            <div className="flex items-center mr-4">
                              <User className="mr-1 h-3 w-3" />
                              {memo.author}
                            </div>
                            <div className="flex items-center">
                              <Calendar className="mr-1 h-3 w-3" />
                              {memo.createdAt}
                            </div>
                          </div>
                        </div>
                        
                        <div className="flex flex-col gap-2 sm:w-auto w-full sm:flex-row">
                          <Button variant="outline" size="sm" className="w-full sm:w-auto">
                            查看
                          </Button>
                          <Button variant="ghost" size="sm" className="w-full sm:w-auto">
                            编辑可见性
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* 可见性说明 */}
      <Card>
        <CardHeader>
          <CardTitle>可见性说明</CardTitle>
          <CardDescription>不同可见性级别的含义</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="border rounded-lg p-4">
              <div className="flex items-center gap-2 mb-3">
                <div className="h-10 w-10 rounded-full bg-green-100 flex items-center justify-center">
                  <Globe className="h-5 w-5 text-green-600" />
                </div>
                <h4 className="font-semibold">公开</h4>
              </div>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• 所有班级成员可见</li>
                <li>• 可以作为示例分享</li>
                <li>• 显示在班级探索页面</li>
              </ul>
            </div>
            
            <div className="border rounded-lg p-4">
              <div className="flex items-center gap-2 mb-3">
                <div className="h-10 w-10 rounded-full bg-yellow-100 flex items-center justify-center">
                  <Eye className="h-5 w-5 text-yellow-600" />
                </div>
                <h4 className="font-semibold">受保护</h4>
              </div>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• 仅班级成员可见</li>
                <li>• 适合内部学习资料</li>
                <li>• 不对外公开</li>
              </ul>
            </div>
            
            <div className="border rounded-lg p-4">
              <div className="flex items-center gap-2 mb-3">
                <div className="h-10 w-10 rounded-full bg-red-100 flex items-center justify-center">
                  <Lock className="h-5 w-5 text-red-600" />
                </div>
                <h4 className="font-semibold">私有</h4>
              </div>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• 仅指定成员可见</li>
                <li>• 适合个人笔记</li>
                <li>• 高度隐私保护</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 分页 */}
      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">
          显示 1-{filteredMemos.length} 条，共 {filteredMemos.length} 条
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" disabled>
            上一页
          </Button>
          <Button variant="outline" size="sm" disabled>
            下一页
          </Button>
        </div>
      </div>
    </div>
  );
};

export default ClassMemosTab;
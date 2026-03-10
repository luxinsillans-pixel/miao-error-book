import { useState } from "react";
import { Plus, Search, Tag, Palette, Edit, Trash2, Copy } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";

import { useClassTagTemplates } from "@/hooks/useClassQueries";

// 表单验证模式
const tagTemplateSchema = z.object({
  displayName: z.string().min(1, "标签名称不能为空").max(50, "名称不能超过50个字符"),
  description: z.string().max(200, "描述不能超过200个字符").optional(),
  color: z.string().regex(/^#[0-9A-Fa-f]{6}$/, "必须是有效的十六进制颜色代码"),
});

type TagTemplateFormValues = z.infer<typeof tagTemplateSchema>;

interface ClassTagTemplatesTabProps {
  classId: string;
}

const ClassTagTemplatesTab = ({ classId }: ClassTagTemplatesTabProps) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<string | null>(null);

  const { data: tagTemplates, isLoading, refetch } = useClassTagTemplates(classId);

  const form = useForm<TagTemplateFormValues>({
    resolver: zodResolver(tagTemplateSchema),
    defaultValues: {
      displayName: "",
      description: "",
      color: "#3B82F6", // 默认蓝色
    },
  });

  // 模拟标签模板数据（实际应该从API获取）
  const mockTagTemplates = [
    {
      id: "1",
      name: "重要概念",
      description: "核心概念和定义",
      color: "#EF4444", // 红色
      usageCount: 42,
    },
    {
      id: "2",
      name: "常见错误",
      description: "学生常犯的错误类型",
      color: "#F59E0B", // 黄色
      usageCount: 28,
    },
    {
      id: "3",
      name: "复习重点",
      description: "考试复习的重点内容",
      color: "#10B981", // 绿色
      usageCount: 35,
    },
    {
      id: "4",
      name: "实验指导",
      description: "实验步骤和注意事项",
      color: "#3B82F6", // 蓝色
      usageCount: 19,
    },
    {
      id: "5",
      name: "扩展阅读",
      description: "额外的学习资源和参考",
      color: "#8B5CF6", // 紫色
      usageCount: 12,
    },
  ];

  const filteredTemplates = mockTagTemplates.filter(template =>
    template.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    template.description.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleCopyColor = (color: string) => {
    navigator.clipboard.writeText(color);
    toast.success("颜色代码已复制");
  };

  const onSubmit = (data: TagTemplateFormValues) => {
    console.log("创建标签模板:", data);
    toast.success("标签模板创建成功");
    setIsCreateDialogOpen(false);
    form.reset();
  };

  const handleEditTemplate = (templateId: string) => {
    const template = mockTagTemplates.find(t => t.id === templateId);
    if (template) {
      form.setValue("displayName", template.name);
      form.setValue("description", template.description);
      form.setValue("color", template.color);
      setEditingTemplate(templateId);
      setIsCreateDialogOpen(true);
    }
  };

  const handleDeleteTemplate = (templateId: string) => {
    if (window.confirm("确定要删除这个标签模板吗？")) {
      toast.success("标签模板已删除");
    }
  };

  const colorPresets = [
    "#EF4444", // 红色
    "#F59E0B", // 黄色
    "#10B981", // 绿色
    "#3B82F6", // 蓝色
    "#8B5CF6", // 紫色
    "#EC4899", // 粉色
    "#6366F1", // 靛蓝色
    "#14B8A6", // 青色
  ];

  return (
    <div className="space-y-6">
      {/* 头部 */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold">标签模板</h2>
          <p className="text-muted-foreground mt-1">
            为班级创建和管理标准的标签模板
          </p>
        </div>
        <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              创建标签模板
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[500px]">
            <DialogHeader>
              <DialogTitle>
                {editingTemplate ? "编辑标签模板" : "创建标签模板"}
              </DialogTitle>
              <DialogDescription>
                创建一个新的标签模板，用于统一班级笔记的分类
              </DialogDescription>
            </DialogHeader>
            
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                <FormField
                  control={form.control}
                  name="displayName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>标签名称 *</FormLabel>
                      <FormControl>
                        <Input placeholder="例如：重要概念" {...field} />
                      </FormControl>
                      <FormDescription>
                        标签的显示名称
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="description"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>描述</FormLabel>
                      <FormControl>
                        <Input 
                          placeholder="标签的简要描述..." 
                          {...field} 
                          value={field.value || ""}
                        />
                      </FormControl>
                      <FormDescription>
                        说明标签的用途和含义
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="color"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>颜色 *</FormLabel>
                      <div className="flex items-center gap-3">
                        <div 
                          className="h-10 w-10 rounded border"
                          style={{ backgroundColor: field.value }}
                        />
                        <FormControl>
                          <Input 
                            placeholder="#3B82F6" 
                            {...field} 
                            className="font-mono"
                          />
                        </FormControl>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => handleCopyColor(field.value)}
                        >
                          <Copy className="h-4 w-4" />
                        </Button>
                      </div>
                      <FormDescription>
                        输入十六进制颜色代码或从预设中选择
                      </FormDescription>
                      
                      <div className="grid grid-cols-8 gap-2 mt-2">
                        {colorPresets.map((color) => (
                          <button
                            key={color}
                            type="button"
                            className="h-8 w-8 rounded border hover:scale-110 transition-transform"
                            style={{ backgroundColor: color }}
                            onClick={() => field.onChange(color)}
                            title={color}
                          />
                        ))}
                      </div>
                      
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <DialogFooter>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setIsCreateDialogOpen(false);
                      form.reset();
                      setEditingTemplate(null);
                    }}
                  >
                    取消
                  </Button>
                  <Button type="submit">
                    {editingTemplate ? "保存更改" : "创建模板"}
                  </Button>
                </DialogFooter>
              </form>
            </Form>
          </DialogContent>
        </Dialog>
      </div>

      {/* 搜索 */}
      <Card>
        <CardContent className="pt-6">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="搜索标签模板..."
              className="pl-10"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
        </CardContent>
      </Card>

      {/* 标签模板列表 */}
      <Card>
        <CardHeader>
          <CardTitle>标签模板列表</CardTitle>
          <CardDescription>
            共 {filteredTemplates.length} 个标签模板
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="p-4 border rounded-lg">
                  <div className="flex items-center gap-3 mb-3">
                    <Skeleton className="h-8 w-8 rounded" />
                    <Skeleton className="h-5 w-32" />
                  </div>
                  <Skeleton className="h-4 w-full mb-2" />
                  <Skeleton className="h-4 w-2/3" />
                </div>
              ))}
            </div>
          ) : filteredTemplates.length === 0 ? (
            <div className="text-center py-12">
              <div className="mx-auto w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-4">
                <Tag className="h-6 w-6 text-muted-foreground" />
              </div>
              <h3 className="text-lg font-semibold mb-2">暂无标签模板</h3>
              <p className="text-muted-foreground mb-4">
                {searchTerm 
                  ? "没有找到匹配的标签模板" 
                  : "当前班级还没有创建标签模板"}
              </p>
              <Button onClick={() => setIsCreateDialogOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                创建第一个标签模板
              </Button>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {filteredTemplates.map((template) => (
                <Card key={template.id} className="hover:shadow-lg transition-shadow">
                  <CardContent className="pt-6">
                    <div className="flex items-start justify-between">
                      <div className="flex items-start gap-3">
                        <div 
                          className="h-10 w-10 rounded-lg flex items-center justify-center"
                          style={{ backgroundColor: template.color }}
                        >
                          <Tag className="h-5 w-5 text-white" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2 mb-1">
                            <h3 className="font-semibold">{template.name}</h3>
                            <Badge variant="outline">
                              使用 {template.usageCount} 次
                            </Badge>
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {template.description}
                          </p>
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEditTemplate(template.id)}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDeleteTemplate(template.id)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                    
                    <div className="flex items-center justify-between mt-4 pt-4 border-t">
                      <div className="flex items-center gap-2">
                        <Palette className="h-4 w-4 text-muted-foreground" />
                        <code className="text-sm font-mono">{template.color}</code>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-6 px-2"
                          onClick={() => handleCopyColor(template.color)}
                        >
                          <Copy className="h-3 w-3" />
                        </Button>
                      </div>
                      <Button variant="outline" size="sm">
                        应用到笔记
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* 使用说明 */}
      <Card>
        <CardHeader>
          <CardTitle>使用说明</CardTitle>
          <CardDescription>如何有效使用标签模板</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div>
              <h4 className="font-semibold mb-2">1. 创建标准标签</h4>
              <p className="text-sm text-muted-foreground">
                为班级创建统一的标签模板，确保所有笔记使用相同的分类标准。
              </p>
            </div>
            
            <div>
              <h4 className="font-semibold mb-2">2. 统一颜色编码</h4>
              <p className="text-sm text-muted-foreground">
                为不同类型的标签分配特定颜色，便于快速识别和筛选。
              </p>
            </div>
            
            <div>
              <h4 className="font-semibold mb-2">3. 批量应用</h4>
              <p className="text-sm text-muted-foreground">
                将标签模板应用到相关笔记，保持分类的一致性和专业性。
              </p>
            </div>
          </div>
          
          <Separator className="my-6" />
          
          <div className="bg-muted p-4 rounded-lg">
            <h4 className="font-semibold mb-2">最佳实践</h4>
            <ul className="text-sm text-muted-foreground space-y-1">
              <li>• 为每个课程单元创建对应的标签模板</li>
              <li>• 使用颜色区分难度级别或重要程度</li>
              <li>• 定期清理不再使用的标签模板</li>
              <li>• 鼓励学生使用标准标签进行笔记分类</li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default ClassTagTemplatesTab;
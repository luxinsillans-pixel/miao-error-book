import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Save, AlertCircle } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useUpdateClass } from "@/hooks/useClassQueries";
import { Class, ClassVisibility } from "@/types/proto/api/v1/class_service_pb";

// 表单验证模式
const settingsSchema = z.object({
  displayName: z.string().min(1, "班级名称不能为空").max(100, "名称不能超过100个字符"),
  description: z.string().max(500, "描述不能超过500个字符").optional(),
  visibility: z.nativeEnum(ClassVisibility),
  settings: z.object({
    studentMemoVisibility: z.boolean().optional(),
    allowAnonymous: z.boolean().optional(),
    enableTagTemplates: z.boolean().optional(),
    maxMembers: z.number().min(0).max(1000).optional(),
    requireMemberApproval: z.boolean().optional(),
  }).optional(),
});

type SettingsFormValues = z.infer<typeof settingsSchema>;

interface ClassSettingsTabProps {
  classId: string;
  classData: Class;
}

const ClassSettingsTab = ({ classId, classData }: ClassSettingsTabProps) => {
  const [isSaving, setIsSaving] = useState(false);
  const updateClassMutation = useUpdateClass();

  const form = useForm<SettingsFormValues>({
    resolver: zodResolver(settingsSchema),
    defaultValues: {
      displayName: classData.displayName,
      description: classData.description,
      visibility: classData.visibility,
      settings: classData.settings || {
        studentMemoVisibility: true,
        allowAnonymous: false,
        enableTagTemplates: true,
        maxMembers: 0,
        requireMemberApproval: false,
      },
    },
  });

  const onSubmit = async (data: SettingsFormValues) => {
    setIsSaving(true);
    try {
      await updateClassMutation.mutateAsync({
        classId,
        class: {
          displayName: data.displayName,
          description: data.description,
          visibility: data.visibility,
          settings: data.settings,
        },
        updateMask: ["display_name", "description", "visibility", "settings"],
      });
      
      toast.success("班级设置已更新");
    } catch (error) {
      toast.error("更新失败");
    } finally {
      setIsSaving(false);
    }
  };

  const getVisibilityDescription = (visibility: ClassVisibility) => {
    switch (visibility) {
      case ClassVisibility.CLASS_PUBLIC:
        return "任何人都可以查看班级，但只有成员可以参与";
      case ClassVisibility.CLASS_PROTECTED:
        return "只有班级成员可以查看和参与";
      case ClassVisibility.CLASS_PRIVATE:
        return "只有受邀成员可以查看和参与";
      default:
        return "";
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">班级设置</h2>
        <p className="text-muted-foreground mt-1">
          管理班级的基本设置和配置
        </p>
      </div>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
          {/* 基本信息 */}
          <Card>
            <CardHeader>
              <CardTitle>基本信息</CardTitle>
              <CardDescription>班级的基本信息和描述</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormField
                control={form.control}
                name="displayName"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>班级名称 *</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>
                      显示给用户的班级名称
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
                    <FormLabel>班级描述</FormLabel>
                    <FormControl>
                      <Textarea 
                        className="min-h-[100px]"
                        {...field} 
                        value={field.value || ""}
                      />
                    </FormControl>
                    <FormDescription>
                      简要描述班级的目的和内容
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>

          {/* 可见性设置 */}
          <Card>
            <CardHeader>
              <CardTitle>可见性设置</CardTitle>
              <CardDescription>控制谁可以查看和访问班级</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormField
                control={form.control}
                name="visibility"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>班级可见性</FormLabel>
                    <Select
                      onValueChange={(value) => field.onChange(parseInt(value))}
                      defaultValue={field.value.toString()}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="选择可见性" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value={ClassVisibility.CLASS_PUBLIC.toString()}>
                          <div>
                            <div className="font-medium">公开</div>
                            <div className="text-xs text-muted-foreground">
                              任何人都可以查看班级
                            </div>
                          </div>
                        </SelectItem>
                        <SelectItem value={ClassVisibility.CLASS_PROTECTED.toString()}>
                          <div>
                            <div className="font-medium">受保护</div>
                            <div className="text-xs text-muted-foreground">
                              只有成员可以查看和参与
                            </div>
                          </div>
                        </SelectItem>
                        <SelectItem value={ClassVisibility.CLASS_PRIVATE.toString()}>
                          <div>
                            <div className="font-medium">私有</div>
                            <div className="text-xs text-muted-foreground">
                              只有受邀成员可以访问
                            </div>
                          </div>
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      {getVisibilityDescription(field.value)}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertTitle>注意</AlertTitle>
                <AlertDescription>
                  更改可见性设置可能会影响现有成员的访问权限。请谨慎操作。
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>

          {/* 高级设置 */}
          <Card>
            <CardHeader>
              <CardTitle>高级设置</CardTitle>
              <CardDescription>班级的高级功能和配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-4">
                <h3 className="font-semibold">笔记设置</h3>
                
                <FormField
                  control={form.control}
                  name="settings.studentMemoVisibility"
                  render={({ field }) => (
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label htmlFor="student-memo-visibility">学生笔记可见</Label>
                        <p className="text-sm text-muted-foreground">
                          允许学生查看彼此的笔记
                        </p>
                      </div>
                      <Switch
                        id="student-memo-visibility"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </div>
                  )}
                />

                <FormField
                  control={form.control}
                  name="settings.allowAnonymous"
                  render={({ field }) => (
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label htmlFor="allow-anonymous">允许匿名提交</Label>
                        <p className="text-sm text-muted-foreground">
                          允许用户匿名提交笔记和问题
                        </p>
                      </div>
                      <Switch
                        id="allow-anonymous"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </div>
                  )}
                />

                <FormField
                  control={form.control}
                  name="settings.enableTagTemplates"
                  render={({ field }) => (
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label htmlFor="enable-tag-templates">启用标签模板</Label>
                        <p className="text-sm text-muted-foreground">
                          为班级启用自定义标签模板
                        </p>
                      </div>
                      <Switch
                        id="enable-tag-templates"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </div>
                  )}
                />
              </div>

              <Separator />

              <div className="space-y-4">
                <h3 className="font-semibold">成员设置</h3>
                
                <FormField
                  control={form.control}
                  name="settings.maxMembers"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>最大成员数</FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          min="0"
                          max="1000"
                          {...field}
                          onChange={(e) => field.onChange(parseInt(e.target.value) || 0)}
                        />
                      </FormControl>
                      <FormDescription>
                        限制班级最大成员数，0表示无限制
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="settings.requireMemberApproval"
                  render={({ field }) => (
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label htmlFor="require-member-approval">需要成员审批</Label>
                        <p className="text-sm text-muted-foreground">
                          新成员加入需要管理员审批
                        </p>
                      </div>
                      <Switch
                        id="require-member-approval"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </div>
                  )}
                />
              </div>
            </CardContent>
          </Card>

          {/* 危险区域 */}
          <Card className="border-red-200">
            <CardHeader>
              <CardTitle className="text-red-600">危险区域</CardTitle>
              <CardDescription className="text-red-500">
                谨慎操作，这些操作可能无法撤销
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between p-4 border border-red-200 rounded-lg">
                <div>
                  <h4 className="font-semibold text-red-600">删除班级</h4>
                  <p className="text-sm text-red-500">
                    永久删除班级及其所有数据
                  </p>
                </div>
                <Button variant="destructive" disabled>
                  删除班级
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* 保存按钮 */}
          <div className="flex justify-end">
            <Button type="submit" disabled={isSaving} size="lg">
              <Save className="mr-2 h-4 w-4" />
              {isSaving ? "保存中..." : "保存设置"}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  );
};

export default ClassSettingsTab;
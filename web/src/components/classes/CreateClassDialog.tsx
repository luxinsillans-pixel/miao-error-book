import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { classServiceClient } from "@/connect";
import { CreateClassRequest, ClassVisibility } from "@/types/proto/api/v1/class_service_pb";
import { toast } from "sonner";

// 表单验证模式
const createClassSchema = z.object({
  classId: z.string().min(1, "班级ID不能为空").regex(/^[a-z0-9-]+$/, "只能包含小写字母、数字和连字符"),
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

type CreateClassFormValues = z.infer<typeof createClassSchema>;

interface CreateClassDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

const CreateClassDialog = ({ open, onOpenChange, onSuccess }: CreateClassDialogProps) => {
  const queryClient = useQueryClient();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const form = useForm<CreateClassFormValues>({
    resolver: zodResolver(createClassSchema),
    defaultValues: {
      classId: "",
      displayName: "",
      description: "",
      visibility: ClassVisibility.CLASS_PUBLIC,
      settings: {
        studentMemoVisibility: true,
        allowAnonymous: false,
        enableTagTemplates: true,
        maxMembers: 0,
        requireMemberApproval: false,
      },
    },
  });

  // 创建班级的Mutation
  const createClassMutation = useMutation({
    mutationFn: async (data: CreateClassFormValues) => {
      const request = new CreateClassRequest();
      request.classId = data.classId;
      
      const cls = {
        name: `classes/${data.classId}`,
        uid: data.classId,
        displayName: data.displayName,
        description: data.description || "",
        visibility: data.visibility,
        settings: data.settings,
      };
      
      request.class = cls;
      
      return await classServiceClient.createClass(request);
    },
    onSuccess: () => {
      toast.success("班级创建成功");
      queryClient.invalidateQueries({ queryKey: ["classes"] });
      onSuccess?.();
      form.reset();
    },
    onError: (error) => {
      toast.error(`创建失败: ${error.message}`);
    },
    onSettled: () => {
      setIsSubmitting(false);
    },
  });

  const onSubmit = (data: CreateClassFormValues) => {
    setIsSubmitting(true);
    createClassMutation.mutate(data);
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!isSubmitting) {
      onOpenChange(newOpen);
      if (!newOpen) {
        form.reset();
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>创建新班级</DialogTitle>
          <DialogDescription>
            创建一个新的班级，用于组织学习资源和错误笔记。
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            {/* 基本设置 */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold">基本设置</h3>
              
              <FormField
                control={form.control}
                name="classId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>班级ID *</FormLabel>
                    <FormControl>
                      <Input 
                        placeholder="my-class-2024" 
                        {...field} 
                        onChange={(e) => field.onChange(e.target.value.toLowerCase())}
                      />
                    </FormControl>
                    <FormDescription>
                      唯一标识符，用于URL中。只能包含小写字母、数字和连字符。
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="displayName"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>班级名称 *</FormLabel>
                    <FormControl>
                      <Input placeholder="2024级计算机科学" {...field} />
                    </FormControl>
                    <FormDescription>
                      显示给用户的班级名称。
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
                        placeholder="描述班级的学习目标和内容..." 
                        className="min-h-[100px]"
                        {...field} 
                        value={field.value || ""}
                      />
                    </FormControl>
                    <FormDescription>
                      简要描述班级的目的和内容。
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            {/* 可见性设置 */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold">可见性设置</h3>
              
              <FormField
                control={form.control}
                name="visibility"
                render={({ field }) => (
                  <FormItem className="space-y-3">
                    <FormLabel>班级可见性</FormLabel>
                    <FormControl>
                      <RadioGroup
                        onValueChange={(value) => field.onChange(parseInt(value))}
                        defaultValue={field.value.toString()}
                        className="flex flex-col space-y-1"
                      >
                        <div className="flex items-center space-x-2">
                          <RadioGroupItem value={ClassVisibility.CLASS_PUBLIC.toString()} id="public" />
                          <Label htmlFor="public" className="cursor-pointer">
                            <div>
                              <p className="font-medium">公开</p>
                              <p className="text-sm text-muted-foreground">
                                任何人都可以查看班级，但只有成员可以参与。
                              </p>
                            </div>
                          </Label>
                        </div>
                        
                        <div className="flex items-center space-x-2">
                          <RadioGroupItem value={ClassVisibility.CLASS_PROTECTED.toString()} id="protected" />
                          <Label htmlFor="protected" className="cursor-pointer">
                            <div>
                              <p className="font-medium">受保护</p>
                              <p className="text-sm text-muted-foreground">
                                只有班级成员可以查看和参与。
                              </p>
                            </div>
                          </Label>
                        </div>
                        
                        <div className="flex items-center space-x-2">
                          <RadioGroupItem value={ClassVisibility.CLASS_PRIVATE.toString()} id="private" />
                          <Label htmlFor="private" className="cursor-pointer">
                            <div>
                              <p className="font-medium">私有</p>
                              <p className="text-sm text-muted-foreground">
                                只有受邀成员可以查看和参与。
                              </p>
                            </div>
                          </Label>
                        </div>
                      </RadioGroup>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            {/* 高级设置 */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold">高级设置</h3>
              
              <div className="space-y-4 border rounded-lg p-4">
                <FormField
                  control={form.control}
                  name="settings.studentMemoVisibility"
                  render={({ field }) => (
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label htmlFor="student-memo-visibility">学生笔记可见</Label>
                        <p className="text-sm text-muted-foreground">
                          允许学生查看彼此的笔记。
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
                          允许用户匿名提交笔记和问题。
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
                          为班级启用自定义标签模板。
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

                <FormField
                  control={form.control}
                  name="settings.requireMemberApproval"
                  render={({ field }) => (
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label htmlFor="require-member-approval">需要成员审批</Label>
                        <p className="text-sm text-muted-foreground">
                          新成员加入需要管理员审批。
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
                          placeholder="0表示无限制"
                          {...field}
                          onChange={(e) => field.onChange(parseInt(e.target.value) || 0)}
                        />
                      </FormControl>
                      <FormDescription>
                        限制班级最大成员数，0表示无限制。
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
                disabled={isSubmitting}
              >
                取消
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? "创建中..." : "创建班级"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};

export default CreateClassDialog;
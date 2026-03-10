import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useMutation } from "@tanstack/react-query";
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
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { useAddClassMember } from "@/hooks/useClassQueries";
import { ClassMemberRole } from "@/types/proto/api/v1/class_service_pb";
import { toast } from "sonner";

// 表单验证模式
const addMemberSchema = z.object({
  userId: z.string().min(1, "用户ID不能为空"),
  role: z.nativeEnum(ClassMemberRole),
  sendInvitation: z.boolean().default(true),
  invitationMessage: z.string().max(500, "邀请消息不能超过500个字符").optional(),
});

type AddMemberFormValues = z.infer<typeof addMemberSchema>;

interface AddMemberDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  classId: string;
  onSuccess?: () => void;
}

const AddMemberDialog = ({ open, onOpenChange, onSuccess, classId }: AddMemberDialogProps) => {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const addMemberMutation = useAddClassMember();

  const form = useForm<AddMemberFormValues>({
    resolver: zodResolver(addMemberSchema),
    defaultValues: {
      userId: "",
      role: ClassMemberRole.STUDENT,
      sendInvitation: true,
      invitationMessage: "",
    },
  });

  const watchSendInvitation = form.watch("sendInvitation");

  const onSubmit = async (data: AddMemberFormValues) => {
    setIsSubmitting(true);
    try {
      await addMemberMutation.mutateAsync({
        classId,
        userId: data.userId,
        role: data.role,
      });
      
      toast.success("成员添加成功");
      form.reset();
      onSuccess?.();
    } catch (error) {
      toast.error("添加失败");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!isSubmitting) {
      onOpenChange(newOpen);
      if (!newOpen) {
        form.reset();
      }
    }
  };

  const getRoleDescription = (role: ClassMemberRole) => {
    switch (role) {
      case ClassMemberRole.TEACHER:
        return "教师可以管理班级设置、成员和所有笔记";
      case ClassMemberRole.ASSISTANT:
        return "助教可以管理学生成员和批改作业";
      case ClassMemberRole.STUDENT:
        return "学生可以提交笔记和参与讨论";
      case ClassMemberRole.PARENT:
        return "家长可以查看班级动态和学生笔记";
      default:
        return "";
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>添加班级成员</DialogTitle>
          <DialogDescription>
            添加新成员到班级。您需要知道用户的ID。
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            <div className="space-y-4">
              <FormField
                control={form.control}
                name="userId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>用户ID *</FormLabel>
                    <FormControl>
                      <Input 
                        placeholder="输入用户ID" 
                        {...field} 
                      />
                    </FormControl>
                    <FormDescription>
                      要添加的用户ID（例如：user123）
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="role"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>成员角色 *</FormLabel>
                    <Select
                      onValueChange={(value) => field.onChange(parseInt(value))}
                      defaultValue={field.value.toString()}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="选择角色" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value={ClassMemberRole.TEACHER.toString()}>
                          <div>
                            <div className="font-medium">教师</div>
                            <div className="text-xs text-muted-foreground">
                              完全管理权限
                            </div>
                          </div>
                        </SelectItem>
                        <SelectItem value={ClassMemberRole.ASSISTANT.toString()}>
                          <div>
                            <div className="font-medium">助教</div>
                            <div className="text-xs text-muted-foreground">
                              协助管理权限
                            </div>
                          </div>
                        </SelectItem>
                        <SelectItem value={ClassMemberRole.STUDENT.toString()}>
                          <div>
                            <div className="font-medium">学生</div>
                            <div className="text-xs text-muted-foreground">
                              标准学习权限
                            </div>
                          </div>
                        </SelectItem>
                        <SelectItem value={ClassMemberRole.PARENT.toString()}>
                          <div>
                            <div className="font-medium">家长</div>
                            <div className="text-xs text-muted-foreground">
                              查看和通知权限
                            </div>
                          </div>
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      {getRoleDescription(field.value)}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className="space-y-4 border rounded-lg p-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label htmlFor="send-invitation">发送邀请</Label>
                    <p className="text-sm text-muted-foreground">
                      向用户发送加入邀请通知
                    </p>
                  </div>
                  <FormField
                    control={form.control}
                    name="sendInvitation"
                    render={({ field }) => (
                      <Switch
                        id="send-invitation"
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    )}
                  />
                </div>

                {watchSendInvitation && (
                  <FormField
                    control={form.control}
                    name="invitationMessage"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>邀请消息</FormLabel>
                        <FormControl>
                          <Textarea
                            placeholder="可选：添加个性化的邀请消息..."
                            className="min-h-[100px]"
                            {...field}
                            value={field.value || ""}
                          />
                        </FormControl>
                        <FormDescription>
                          这将包含在发送给用户的邀请中
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
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
                {isSubmitting ? "添加中..." : "添加成员"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};

export default AddMemberDialog;
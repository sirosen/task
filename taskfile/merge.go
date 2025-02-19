package taskfile

import (
	"fmt"
	"strings"

	"github.com/go-task/task/v3/taskfile/ast"
)

// NamespaceSeparator contains the character that separates namespaces
const NamespaceSeparator = ":"

// Merge merges the second Taskfile into the first
func Merge(t1, t2 *ast.Taskfile, include *ast.Include) error {
	if !t1.Version.Equal(t2.Version) {
		return fmt.Errorf(`task: Taskfiles versions should match. First is "%s" but second is "%s"`, t1.Version, t2.Version)
	}
	if t2.Output.IsSet() {
		t1.Output = t2.Output
	}

	if t1.Vars == nil {
		t1.Vars = &ast.Vars{}
	}
	if t1.Env == nil {
		t1.Env = &ast.Vars{}
	}
	t1.Vars.Merge(t2.Vars)
	t1.Env.Merge(t2.Env)

	if err := t2.Tasks.Range(func(k string, v *ast.Task) error {
		// We do a deep copy of the task struct here to ensure that no data can
		// be changed elsewhere once the taskfile is merged.
		task := v.DeepCopy()

		// Set the task to internal if EITHER the included task or the included
		// taskfile are marked as internal
		task.Internal = task.Internal || (include != nil && include.Internal)

		// Add namespaces to dependencies, commands and aliases
		for _, dep := range task.Deps {
			if dep != nil && dep.Task != "" {
				dep.Task = taskNameWithNamespace(dep.Task, include.Namespace)
			}
		}
		for _, cmd := range task.Cmds {
			if cmd != nil && cmd.Task != "" {
				cmd.Task = taskNameWithNamespace(cmd.Task, include.Namespace)
			}
		}
		for i, alias := range task.Aliases {
			task.Aliases[i] = taskNameWithNamespace(alias, include.Namespace)
		}
		// Add namespace aliases
		if include != nil {
			for _, namespaceAlias := range include.Aliases {
				task.Aliases = append(task.Aliases, taskNameWithNamespace(task.Task, namespaceAlias))
				for _, alias := range v.Aliases {
					task.Aliases = append(task.Aliases, taskNameWithNamespace(alias, namespaceAlias))
				}
			}
		}

		// Add the task to the merged taskfile
		taskNameWithNamespace := taskNameWithNamespace(k, include.Namespace)
		task.Task = taskNameWithNamespace
		t1.Tasks.Set(taskNameWithNamespace, task)

		return nil
	}); err != nil {
		return err
	}

	// If the included Taskfile has a default task and the parent namespace has
	// no task with a matching name, we can add an alias so that the user can
	// run the included Taskfile's default task without specifying its full
	// name. If the parent namespace has aliases, we add another alias for each
	// of them.
	if t2.Tasks.Get("default") != nil && t1.Tasks.Get(include.Namespace) == nil {
		defaultTaskName := fmt.Sprintf("%s:default", include.Namespace)
		t1.Tasks.Get(defaultTaskName).Aliases = append(t1.Tasks.Get(defaultTaskName).Aliases, include.Namespace)
		t1.Tasks.Get(defaultTaskName).Aliases = append(t1.Tasks.Get(defaultTaskName).Aliases, include.Aliases...)
	}

	return nil
}

func taskNameWithNamespace(taskName string, namespace string) string {
	if strings.HasPrefix(taskName, NamespaceSeparator) {
		return strings.TrimPrefix(taskName, NamespaceSeparator)
	}
	return fmt.Sprintf("%s%s%s", namespace, NamespaceSeparator, taskName)
}

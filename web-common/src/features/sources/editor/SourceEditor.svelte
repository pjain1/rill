<script lang="ts">
  import type { EditorView } from "@codemirror/view";
  import type { V1ParseError } from "@rilldata/web-common/runtime-client";
  import { setLineStatuses } from "../../../components/editor/line-status";
  import { mapParseErrorsToLines } from "../../metrics-views/errors";
  import Editor from "../../editor/Editor.svelte";
  import { FileExtensionToEditorExtension } from "../../editor/getExtensionsForFile";
  import { FileArtifact } from "../../entity-management/file-artifact";

  export let allErrors: V1ParseError[];
  export let fileArtifact: FileArtifact;
  export let onSave: () => void = () => {};

  $: ({ remoteContent, localContent } = fileArtifact);

  let editor: EditorView;

  function handleUpdate() {
    onSave();
    // Clear line errors (it's confusing when they're outdated)
    setLineStatuses([], editor);
  }

  //  Handle errors
  $: if (editor)
    setLineStatuses(
      mapParseErrorsToLines(allErrors, $localContent ?? $remoteContent ?? ""),
      editor,
    );
</script>

<div class="editor flex flex-col border border-gray-200 rounded h-full">
  <div class="grow flex bg-white overflow-y-auto rounded">
    <Editor
      {fileArtifact}
      extensions={FileExtensionToEditorExtension[".yaml"]}
      bind:editor
      forceDisableAutoSave
      onSave={handleUpdate}
    />
  </div>
</div>

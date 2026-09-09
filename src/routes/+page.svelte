<script lang="ts">
	import { Chat } from '@ai-sdk/svelte';
	import { DefaultChatTransport, type ToolUIPart } from 'ai';

	let input = $state('');
	const { data } = $props();
	let messages = $derived(data.initialMessages);

	const chat = new Chat({
		transport: new DefaultChatTransport({
			api: '/api/chat'
		}),
		get messages() {
			return messages;
		}
	});

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		chat.sendMessage({ text: input });
		input = '';
	}

	const STATE_TO_LABEL_MAP: Record<ToolUIPart['state'], string> = {
		'input-streaming': 'Pending',
		'input-available': 'Running',
		'approval-requested': 'Awaiting Approval',
		'approval-responded': 'Responded',
		'output-available': 'Completed',
		'output-error': 'Error',
		'output-denied': 'Denied'
	};
</script>

<main class="mx-auto size-full h-screen max-w-3xl p-6">
	<div class="flex h-full flex-col">
		<div class="min-h-0 flex-1 overflow-y-auto" data-name="conversation">
			<div data-name="conversation-content" class="flex flex-col gap-8">
				{#each chat.messages as message, messageIndex (messageIndex)}
					<div>
						{#each message.parts as part, partIndex (partIndex)}
							{#if part.type === 'text'}
								<div
									data-name="message"
									class={[
										message.role === 'user' && 'ml-auto justify-end',
										'group flex w-full max-w-[95%] flex-col gap-2',
										message.role === 'user' ? 'is-user' : 'is-assistant'
									]}
								>
									<div
										data-name="message-content"
										class={[
											'is-user:dark flex w-fit max-w-full min-w-0 flex-col gap-2 overflow-hidden text-sm',
											'group-[.is-user]:text-foreground group-[.is-user]:ml-auto group-[.is-user]:rounded-lg group-[.is-user]:bg-blue-100 group-[.is-user]:px-4 group-[.is-user]:py-3',
											'group-[.is-assistant]:text-foreground'
										]}
									>
										<div
											data-name="message-response"
											class="size-full [&>*:first-child]:mt-0 [&>*:last-child]:mb-0"
										>
											{part.text}
										</div>
									</div>
								</div>
							{:else if part.type.startsWith('tool-')}
								<div
									data-name="tool"
									class="not-prose mb-6 w-full rounded-lg border border-gray-300 shadow"
								>
									<details data-name="tool-header" class="w-full p-3 hover:cursor-pointer">
										<summary class="text-sm font-medium"
											>{(part as ToolUIPart).type.split('-').slice(1).join('-')} - {STATE_TO_LABEL_MAP[
												(part as ToolUIPart).state ?? 'output-available'
											]}</summary
										>
										<div data-name="tool-content" class="">
											<div data-name="tool-input" class="space-y-2 overflow-hidden py-4">
												<div
													class="text-muted-foreground text-xs font-medium tracking-wide uppercase"
												>
													Parameters
												</div>
												<pre
													class="w-full overflow-x-auto rounded-md border border-gray-300 bg-gray-50 p-3 text-sm"><code
														>{JSON.stringify((part as ToolUIPart).input, null, 2)}</code
													></pre>
											</div>
											<div data-name="tool-output" class="space-y-2 overflow-hidden py-4">
												<div
													class="text-muted-foreground text-xs font-medium tracking-wide uppercase"
												>
													{(part as ToolUIPart).errorText ? 'Error' : 'Result'}
												</div>
												<pre
													class="w-full overflow-x-auto rounded-md border border-gray-300 bg-gray-50 p-3 text-sm"><code
														>{JSON.stringify((part as ToolUIPart).output, null, 2)}</code
													></pre>
												{#if (part as ToolUIPart).errorText}
													<div data-name="tool-error" class="text-red-600">
														{(part as ToolUIPart).errorText}
													</div>
												{/if}
											</div>
										</div>
									</details>
								</div>
							{/if}
						{/each}
					</div>
				{/each}
			</div>
		</div>
		<form
			class="grid w-full shrink-0 grid-cols-[1fr_auto] gap-6 pt-4"
			onsubmit={handleSubmit}
			data-name="prompt-input"
		>
			<input
				name="chat-input"
				class="h-10 rounded-lg border border-gray-300 shadow"
				placeholder="City name"
				bind:value={input}
			/>
			<button
				class="focus-visible:border-ring focus-visible:ring-ring/50 shrink-0 rounded-lg border border-blue-400 bg-blue-600 px-4 text-sm font-medium whitespace-nowrap text-white shadow-lg transition-all outline-none focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50"
				type="submit">Send</button
			>
		</form>
	</div>
</main>

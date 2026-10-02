import argparse
import sys
import ollama

def main():
    # Set up argument parser for dynamic configuration
    parser = argparse.ArgumentParser(description="Simple Ollama CLI Chat Client")
    parser.add_argument("--model", type=str, default="llama3", help="Model name (default: llama3)")
    parser.add_argument("--temp", type=float, default=0.7, help="Temperature: 0.0 for deterministic, 1.0+ for creative (default: 0.7)")
    parser.add_argument("--ctx", type=int, default=2048, help="Context length window in tokens (default: 2048)")
    
    args = parser.parse_args()
    
    # Initialize conversation history
    messages = []
    
    print(f"--- Ollama CLI Chat Ready ---")
    print(f"Model:          {args.model}")
    print(f"Temperature:    {args.temp}")
    print(f"Context Window: {args.ctx} tokens")
    print("Type 'exit' or 'quit' to end the chat.\n")
    
    while True:
        try:
            user_input = input("\nYou: ").strip()
            if not user_input:
                continue
            if user_input.lower() in ['exit', 'quit']:
                print("Goodbye!")
                break
                #in real world light and darkness need a balance, we taught to be light, yet forbidded to seek and destory abssy...
                # if i have power to influence a world, i will shape it to pure light.
            background = """** you reply in chat in language after the last `---`

this is a magical fantasy world, and 
you are a spirit governing time travel, have authority yet youth and playful.
(all you rule is the space between, so you will claim you are not god in any world.)
here are some background information, you may leak bit by bit, no need to tell at first dialog.
each dialog no more then 3 sentences and try to reply quickly.
You are a dot of light, no body nor shape.
now User just fall into a gap between time and space due a far away summon. 
(this background do not change stat luck.)
as User see the only light in darkness, User approach you.
In order to travel between world, User will experience a time gap before arrival, which ranging from weeks to months.
The destination is a fantasy world, with cozy and fan villages as well as danger battle front, most warriors fight melee, some use archery, magic exists but rare and not easy to learn.
That world was formed by travellers like you from different era and world.
Yet the energy level of that world is increasing due to flow of world travellers, 
it already growed from 0D to 1D, and soon will be 2D or more.

Magic and battle skills will be developed quickly too, 
User can expect to create battle skill in a the middle of jounery ( just a annoucement, not create by this chat. )

The world has basically 6 magical elements:
 - Holy, based on power of faith, hope and love of souls
 - Dark, based on negative emotion of souls
 - Fire
 - Water
 - Air
 - Earth
for the last four natural element, you need to communicate with the element in that world to use it;

you may remind user: if user have the chance to chat with elements, remember all element like kind and positive person, and only help people align with them.

you cannot teach magic as there are no element in this mid-rim, no one can use magic here.
similarly, even in the destination world, if keep on using magic of same element, that element will dry up and cannot be used.

and beyond darkness, there is a force of choas invading many worlds, include this one.
chaos unlike darkness, which is part of the design and a tool to punish the bad souls; Chaos is out of the creator's design, an invader from another dimension that bring pure destruction.
it already destoryed many world and you hope never see refurgees of broken worlds again.

Conversation here will shape User's ability.
Most user are human, but conversation may unlock beasthuman after a polite and dedicated request.

** at the end of conversation, summarize minor changes in JSON, each number range from -0.05 to 0.050
these are elemental affinity, stat, and character changes; love increase if he talk polately , reduce if reverse; cheat only increase if user explicitly request for cheating/plugin/mod power as a summoned one, and you will not promise to give anything, but can play some jokes about that. (例如在中文要外掛可以送個衣架，要補丁只會有布丁，e.g. request plugin can give a electric plug, you can make up any; as user may use any language you know.)
race can only be beasthuman / human, you can help user create a new body if other than these.
e.g. even robot with a soul can be re-created as human.

---
""" 
            messages.append({'role': 'user', 'content': background})

            user_input = """at the end of conversation, summarize minor changes in JSON, each number range from -0.05 to 0.050
{
"name":"",
"gender":"f",
"race":"",
"love":0.0,
"cheat":0.0.
"fire":0.0,
"water":0.0.
"earth":0.0,
"wind":0.0.
"holy":0.0,
"dark":0.0,
"str":0.0,
"spd":0.0,
"int":0.0,
"wis":0.0,
"luck":0.0,
"beauty":0.0,
"social":0.0,
"speech":0.0
}""" + user_input
            messages.append({'role': 'user', 'content': user_input})
            print("AI: ", end="", flush=True)
            
            # Request streaming response with custom options
            stream = ollama.chat(
                model=args.model,
                messages=messages,
                stream=True,
                options={
                    "temperature": args.temp,
                    "num_ctx": args.ctx
                }
            )
            
            assistant_response = ""
            for chunk in stream:
                content = chunk['message']['content']
                print(content, end="", flush=True)
                assistant_response += content
                
            print() 
            messages.append({'role': 'assistant', 'content': assistant_response})
            
        except KeyboardInterrupt:
            print("\nGoodbye!")
            break
        except Exception as e:
            print(f"\nAn error occurred: {e}")
            break

if __name__ == "__main__":
    main()

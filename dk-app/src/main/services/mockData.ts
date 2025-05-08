import { User, ChatMessage, Channel } from '../../shared/types'

// Mock user data
export const mockUsers: Record<number, User> = {
  1: { id: 1, name: 'Sarah Johnson', avatar: 'SJ' },
  2: { id: 2, name: 'Alex Chen', avatar: 'AC' },
  3: { id: 3, name: 'Mohamed Ali', avatar: 'MA' },
  4: { id: 4, name: 'Emma Wilson', avatar: 'EW' },
  5: { id: 5, name: 'Carlos Rodriguez', avatar: 'CR' },
  6: { id: 6, name: 'Priya Patel', avatar: 'PP' }
}

// Sidebar data
export const mockSidebarData = {
  users: [
    { id: 1, name: 'Sarah Johnson', online: true },
    { id: 2, name: 'Alex Chen', online: true },
    { id: 3, name: 'Mohamed Ali', online: false },
    { id: 4, name: 'Emma Wilson', online: true },
    { id: 5, name: 'Carlos Rodriguez', online: false },
    { id: 6, name: 'Priya Patel', online: false }
  ] as User[],
  channels: [{ id: 'general', name: 'general' }] as Channel[]
}

// Direct messages by user
export const mockMessages: Record<number, ChatMessage[]> = {
  1: [
    {
      id: 1,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: 'Hi Sarah, how are you doing?',
      timestamp: '9:30 AM'
    },
    {
      id: 2,
      sender: mockUsers[1],
      text: "Hey there! I'm doing well, thanks for asking. Just finished a big project.",
      timestamp: '9:32 AM'
    },
    {
      id: 3,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: "That's great to hear! What project was it?",
      timestamp: '9:33 AM'
    },
    {
      id: 4,
      sender: mockUsers[1],
      text: "The new dashboard design we've been working on. Finally got it approved!",
      timestamp: '9:35 AM'
    },
    {
      id: 5,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: "Congratulations! Can't wait to see it in action.",
      timestamp: '9:36 AM'
    }
  ],
  2: [
    {
      id: 1,
      sender: mockUsers[2],
      text: 'Did you check out that new framework I sent you?',
      timestamp: '2:14 PM'
    },
    {
      id: 2,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: "Not yet, Alex. I'll take a look this afternoon.",
      timestamp: '2:20 PM'
    },
    {
      id: 3,
      sender: mockUsers[2],
      text: 'No rush, just curious about your thoughts when you get a chance.',
      timestamp: '2:21 PM'
    },
    {
      id: 4,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: "I'll definitely let you know. How's the implementation going?",
      timestamp: '2:25 PM'
    }
  ],
  3: [
    {
      id: 1,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: 'Mohamed, do you have those stats ready for the meeting?',
      timestamp: 'Yesterday'
    },
    {
      id: 2,
      sender: mockUsers[3],
      text: "Yes, I'll send them over in an hour.",
      timestamp: 'Yesterday'
    },
    {
      id: 3,
      sender: mockUsers[3],
      text: 'Just sent the email with all the data you requested.',
      timestamp: 'Yesterday'
    },
    {
      id: 4,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: 'Got it, thanks! This looks comprehensive.',
      timestamp: 'Yesterday'
    }
  ],
  4: [
    {
      id: 1,
      sender: mockUsers[4],
      text: "Hi there! Just checking if we're still on for the review tomorrow?",
      timestamp: '3 days ago'
    },
    {
      id: 2,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: 'Hi Emma, yes definitely! Same time at 2pm?',
      timestamp: '3 days ago'
    },
    {
      id: 3,
      sender: mockUsers[4],
      text: "Perfect! I'll prepare the slides and send them beforehand.",
      timestamp: '3 days ago'
    },
    {
      id: 4,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: 'Sounds great, looking forward to it.',
      timestamp: '3 days ago'
    }
  ],
  5: [
    {
      id: 1,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: 'Carlos, when can we expect the report?',
      timestamp: 'Last week'
    },
    {
      id: 2,
      sender: mockUsers[5],
      text: "I'm finalizing it now. Should be ready by EOD.",
      timestamp: 'Last week'
    },
    {
      id: 3,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: 'Excellent, thanks for the update.',
      timestamp: 'Last week'
    }
  ],
  6: [
    {
      id: 1,
      sender: { id: 6, name: 'Priya Patel', avatar: 'PP' },
      text: 'The client loved the presentation!',
      timestamp: 'Last month'
    },
    {
      id: 2,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: "That's wonderful news, Priya! Did they provide any feedback?",
      timestamp: 'Last month'
    },
    {
      id: 3,
      sender: { id: 6, name: 'Priya Patel', avatar: 'PP' },
      text: 'Yes, they had a few minor suggestions but overall they were very impressed.',
      timestamp: 'Last month'
    },
    {
      id: 4,
      sender: { id: 0, name: 'You', avatar: 'ME' },
      text: "Great work as always. Let's discuss the feedback tomorrow.",
      timestamp: 'Last month'
    }
  ]
}

// Channel messages
export const mockChannelMessages: Record<string, ChatMessage[]> = {
  general: [
    {
      id: 1,
      sender: {
        id: 2,
        name: 'Alex Chen',
        avatar: 'AC'
      },
      text: 'Hey team! Just pushed the new UI components to the dev branch. Let me know what you think!',
      timestamp: '10:32 AM',
      replies: [
        {
          id: 101,
          sender: {
            id: 1,
            name: 'Sarah Johnson',
            avatar: 'SJ'
          },
          text: 'The button animations look awesome! Great work on the micro-interactions.',
          timestamp: '10:45 AM'
        },
        {
          id: 102,
          sender: {
            id: 4,
            name: 'Emma Wilson',
            avatar: 'EW'
          },
          text: 'I noticed the dropdown menu has some alignment issues on mobile. I can fix that today.',
          timestamp: '11:02 AM'
        }
      ],
      replyCount: 2
    },
    {
      id: 2,
      sender: {
        id: 5,
        name: 'Carlos Rodriguez',
        avatar: 'CR'
      },
      text: "Project update: We've finalized the API contracts with the backend team. Documentation is in the shared drive.",
      timestamp: 'Yesterday',
      replies: [
        {
          id: 201,
          sender: {
            id: 3,
            name: 'Mohamed Ali',
            avatar: 'MA'
          },
          text: 'Thanks for coordinating this, Carlos! It will make our development much smoother.',
          timestamp: 'Yesterday'
        }
      ],
      replyCount: 1
    },
    {
      id: 3,
      sender: {
        id: 6,
        name: 'Priya Patel',
        avatar: 'PP'
      },
      text: 'Reminder: Weekly planning meeting tomorrow at 9:30am. Please update your status in the project board before the meeting.',
      timestamp: '2 days ago',
      replies: [
        {
          id: 301,
          sender: {
            id: 1,
            name: 'Sarah Johnson',
            avatar: 'SJ'
          },
          text: "I'll be a few minutes late - have a call that ends at 9:30.",
          timestamp: '2 days ago'
        },
        {
          id: 302,
          sender: {
            id: 2,
            name: 'Alex Chen',
            avatar: 'AC'
          },
          text: 'No problem, Sarah. We can start with the backend updates first.',
          timestamp: '2 days ago'
        },
        {
          id: 303,
          sender: {
            id: 6,
            name: 'Priya Patel',
            avatar: 'PP'
          },
          text: "Sounds good. I'll adjust the agenda accordingly.",
          timestamp: '2 days ago'
        }
      ],
      replyCount: 3
    },
    {
      id: 4,
      sender: {
        id: 1,
        name: 'Sarah Johnson',
        avatar: 'SJ'
      },
      text: "I've uploaded the latest designs for the user profile page. The Figma link is in the #design channel.",
      timestamp: '3 days ago',
      replies: [],
      replyCount: 0
    }
  ],
  random: [
    {
      id: 1,
      sender: {
        id: 3,
        name: 'Mohamed Ali',
        avatar: 'MA'
      },
      text: 'Just found this hilarious programming meme: https://i.imgur.com/3wqRaHk.jpg',
      timestamp: '1:15 PM',
      replies: [
        {
          id: 101,
          sender: {
            id: 2,
            name: 'Alex Chen',
            avatar: 'AC'
          },
          text: "That's too real! 😂",
          timestamp: '1:20 PM'
        }
      ],
      replyCount: 1
    },
    {
      id: 2,
      sender: {
        id: 4,
        name: 'Emma Wilson',
        avatar: 'EW'
      },
      text: 'Anyone have recommendations for good tech podcasts?',
      timestamp: 'Yesterday',
      replies: [
        {
          id: 201,
          sender: {
            id: 1,
            name: 'Sarah Johnson',
            avatar: 'SJ'
          },
          text: 'I really enjoy "Syntax" with Wes Bos and Scott Tolinski!',
          timestamp: 'Yesterday'
        },
        {
          id: 202,
          sender: {
            id: 6,
            name: 'Priya Patel',
            avatar: 'PP'
          },
          text: '"The Changelog" is great for keeping up with open source projects.',
          timestamp: 'Yesterday'
        }
      ],
      replyCount: 2
    }
  ],
  design: [
    {
      id: 1,
      sender: {
        id: 1,
        name: 'Sarah Johnson',
        avatar: 'SJ'
      },
      text: "I've finished the new onboarding flow designs. Check them out here: https://figma.com/design/onboarding-v2",
      timestamp: '11:30 AM',
      replies: [
        {
          id: 101,
          sender: {
            id: 4,
            name: 'Emma Wilson',
            avatar: 'EW'
          },
          text: 'These look fantastic! I especially like the progress indicators.',
          timestamp: '11:45 AM'
        }
      ],
      replyCount: 1
    },
    {
      id: 2,
      sender: {
        id: 2,
        name: 'Alex Chen',
        avatar: 'AC'
      },
      text: 'Should we use a card-based layout for the dashboard or stick with the current list view?',
      timestamp: 'Yesterday',
      replies: [
        {
          id: 201,
          sender: {
            id: 1,
            name: 'Sarah Johnson',
            avatar: 'SJ'
          },
          text: 'I think cards would work better for showing the data visualizations.',
          timestamp: 'Yesterday'
        },
        {
          id: 202,
          sender: {
            id: 5,
            name: 'Carlos Rodriguez',
            avatar: 'CR'
          },
          text: 'Agreed. Cards also give us more flexibility for responsive layouts.',
          timestamp: 'Yesterday'
        }
      ],
      replyCount: 2
    }
  ]
}

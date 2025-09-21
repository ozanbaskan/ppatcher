#include <stdio.h>
#include <unistd.h>

int main() {
    printf("Test program started!\n");
    for (int i = 0; i < 5; i++) {
        printf("Working... %d\n", i);
        sleep(1);
    }
    printf("Test program finished!\n");
    return 0;
}

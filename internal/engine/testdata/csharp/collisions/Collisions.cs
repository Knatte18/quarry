namespace Loomyard.Collisions;

public partial record Shape(int Width);

public partial record Shape
{
    public int Width { get; init; }
}

public record Parent
{
    public int Id { get; init; }
}

public record Child(int Id) : Parent;

public class Renderer : Renderer.IHandler
{
    public interface IHandler
    {
        void Handle();
    }

    void IHandler.Handle() { }
}
